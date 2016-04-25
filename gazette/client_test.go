package gazette

import (
	"bytes"
	"errors"
	"expvar"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	gc "github.com/go-check/check"
	"github.com/stretchr/testify/mock"

	. "github.com/pippio/gazette/journal"
)

const (
	kFragmentFixtureStr = "00000000000003e8-00000000000007d0-" +
		"0102030405060708090a0b0c0d0e0f1011121314"
)

var fragmentFixture = Fragment{
	Journal: "a/journal",
	Begin:   1000,
	End:     2000,
	Sum:     [...]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
}

type ClientSuite struct {
	client *Client
}

func (s *ClientSuite) SetUpTest(c *gc.C) {
	// Clear the normally global stats-map for each test.
	gazetteMap.Init()

	client, err := NewClient("http://default")
	c.Assert(err, gc.IsNil)
	client.timeNow = func() time.Time { return time.Unix(1234, 0) } // Fix time.
	s.client = client
}

func newReadResponseFixture() *http.Response {
	return &http.Response{
		// Return a successful HEAD response, which was redirected from http://default.
		StatusCode: http.StatusPartialContent,
		Header: http.Header{
			"Content-Range":        []string{"bytes 1005-9999999999/9999999999"},
			WriteHeadHeader:        []string{"3000"},
			FragmentNameHeader:     []string{kFragmentFixtureStr},
			FragmentLocationHeader: []string{"http://cloud/fragment/location"},
		},
		Request: &http.Request{
			URL: newURL("http://redirected-server/a/journal"),
		},
		Body: ioutil.NopCloser(strings.NewReader("body")),
	}
}

func (s *ClientSuite) TestHeadRequest(c *gc.C) {
	mockClient := &mockHttpClient{}

	mockClient.On("Do", mock.MatchedBy(func(request *http.Request) bool {
		return request.Method == "HEAD" &&
			request.URL.String() == "http://default/a/journal?block=false&offset=1005"
	})).Return(newReadResponseFixture(), nil).Once()

	s.client.httpClient = mockClient
	result, loc := s.client.Head(ReadArgs{Journal: "a/journal", Offset: 1005, Blocking: false})

	c.Check(result, gc.DeepEquals, ReadResult{
		Offset:    1005,
		WriteHead: 3000,
		Fragment:  fragmentFixture,
	})
	c.Check(loc, gc.DeepEquals, newURL("http://cloud/fragment/location"))

	mockClient.AssertExpectations(c)

	// Expect that the redirected location was cached.
	cached, _ := s.client.locationCache.Get("/a/journal")
	c.Check(cached, gc.DeepEquals, newURL("http://redirected-server/a/journal"))
}

func (s *ClientSuite) TestDirectGet(c *gc.C) {
	mockClient := &mockHttpClient{}

	responseFixture := newReadResponseFixture()
	mockClient.On("Do", mock.MatchedBy(func(request *http.Request) bool {
		return request.Method == "GET" &&
			request.URL.String() == "http://default/a/journal?block=false&offset=1005"
	})).Return(responseFixture, nil).Once()

	s.client.httpClient = mockClient
	result, body := s.client.GetDirect(ReadArgs{
		Journal: "a/journal", Offset: 1005, Blocking: false})

	c.Check(result, gc.DeepEquals, ReadResult{
		Offset:    1005,
		WriteHead: 3000,
		Fragment:  fragmentFixture,
	})
	mockClient.AssertExpectations(c)

	// Expect server's response body is plugged into the stats-wrapper.
	c.Check(body.(readStatsWrapper).stream, gc.Equals, responseFixture.Body)
}

func (s *ClientSuite) TestDirectGetFails(c *gc.C) {
	mockClient := &mockHttpClient{}

	mockClient.On("Do", mock.MatchedBy(func(request *http.Request) bool {
		return request.Method == "GET" &&
			request.URL.String() == "http://default/a/journal?block=false&offset=1005"
	})).Return(&http.Response{
		StatusCode: http.StatusInternalServerError,
		Status:     "Internal Error",
		Body:       ioutil.NopCloser(strings.NewReader("message")),
	}, nil).Once()

	s.client.httpClient = mockClient
	result, body := s.client.GetDirect(ReadArgs{
		Journal: "a/journal", Offset: 1005, Blocking: false})

	c.Check(result.Error.Error(), gc.Equals, "Internal Error (message)")
	c.Check(body, gc.IsNil)

	mockClient.AssertExpectations(c)
}

func (s *ClientSuite) TestGetWithoutFragmentLocation(c *gc.C) {
	mockClient := &mockHttpClient{}

	responseFixture := newReadResponseFixture()
	responseFixture.Header.Del(FragmentLocationHeader)

	// Expect an initial non-blocking HEAD request to the default endpoint.
	mockClient.On("Do", mock.MatchedBy(func(request *http.Request) bool {
		return request.Method == "HEAD" &&
			request.URL.String() == "http://default/a/journal?block=false&offset=1005"
	})).Return(responseFixture, nil).Once()

	// Expect a direct blocking GET request to the previously-redirected endpoint.
	mockClient.On("Do", mock.MatchedBy(func(request *http.Request) bool {
		return request.Method == "GET" &&
			request.URL.String() == "http://redirected-server/a/journal?block=true&blockms=6000&offset=1005"
	})).Return(responseFixture, nil).Once()

	// Arbitrary offset fixture.
	s.client.httpClient = mockClient
	result, body := s.client.Get(ReadArgs{
		Journal: "a/journal", Offset: 1005, Blocking: true, Deadline: time.Unix(1240, 0)})

	c.Check(result, gc.DeepEquals, ReadResult{
		Offset:    1005,
		WriteHead: 3000,
		Fragment:  fragmentFixture,
	})
	mockClient.AssertExpectations(c)

	// Expect server's response body is plugged into the stats-wrapper.
	c.Check(body.(readStatsWrapper).stream, gc.Equals, responseFixture.Body)

	// Before the read, head is at the requested offset.
	readerMap := gazetteMap.Get("readers").(*expvar.Map).Get("a/journal").(*expvar.Map)
	c.Check(readerMap.Get("head").(*expvar.Int).String(), gc.Equals, "1005")

	// Read the data out, so our stats-wrapper sees it ("body" -- 4 bytes)
	io.Copy(ioutil.Discard, body)

	// After the read, byte counter goes up and offset advances with the size
	// of the read.
	c.Check(readerMap.Get("bytes").(*expvar.Int).String(), gc.Equals, "4")
	c.Check(readerMap.Get("head").(*expvar.Int).String(), gc.Equals, "1009")
}

func (s *ClientSuite) TestGetWithFragmentLocation(c *gc.C) {
	mockClient := &mockHttpClient{}

	// Expect an initial HEAD request to the default endpoint.
	mockClient.On("Do", mock.MatchedBy(func(request *http.Request) bool {
		return request.Method == "HEAD" &&
			request.URL.String() == "http://default/a/journal?block=false&offset=1005"
	})).Return(newReadResponseFixture(), nil).Once()

	// Expect a following GET request to the returned cloud URL.
	mockClient.On("Get", "http://cloud/fragment/location").Return(&http.Response{
		StatusCode: http.StatusOK,
		Body:       ioutil.NopCloser(strings.NewReader("xxxxxfragment-content...")),
	}, nil).Once()

	s.client.httpClient = mockClient
	result, body := s.client.Get(ReadArgs{Journal: "a/journal", Offset: 1005, Blocking: false})

	c.Check(result.Error, gc.IsNil)
	c.Check(result, gc.DeepEquals, ReadResult{
		Offset:    1005,
		WriteHead: 3000,
		Fragment:  fragmentFixture,
	})
	mockClient.AssertExpectations(c)

	// Expect that the returned response is pre-seeked to the correct offset.
	data, _ := ioutil.ReadAll(body)
	c.Check(string(data), gc.Equals, "fragment-content...")
}

func (s *ClientSuite) TestGetWithFragmentLocationFails(c *gc.C) {
	mockClient := &mockHttpClient{}

	// Expect an initial HEAD request to the default endpoint.
	mockClient.On("Do", mock.MatchedBy(func(request *http.Request) bool {
		return request.Method == "HEAD" &&
			request.URL.String() == "http://default/a/journal?block=false&offset=1005"
	})).Return(newReadResponseFixture(), nil).Once()

	// Expect a following GET request to the returned cloud URL, which fails.
	mockClient.On("Get", "http://cloud/fragment/location").Return(&http.Response{
		StatusCode: http.StatusInternalServerError,
		Status:     "Internal Error",
		Body:       ioutil.NopCloser(strings.NewReader("message")),
	}, nil).Once()

	s.client.httpClient = mockClient
	result, body := s.client.Get(ReadArgs{Journal: "a/journal", Offset: 1005, Blocking: false})

	c.Check(result.Error.Error(), gc.Equals, "fetching fragment: Internal Error")
	c.Check(body, gc.IsNil)

	mockClient.AssertExpectations(c)
}

func (s *ClientSuite) TestGetPersistedErrorCases(c *gc.C) {
	mockClient := &mockHttpClient{}
	s.client.httpClient = mockClient

	location := newURL("http://cloud/location")
	readResult := ReadResult{Offset: 1005, WriteHead: 3000, Fragment: fragmentFixture}

	// Expect response errors are passed through.
	mockClient.On("Get", "http://cloud/location").Return(nil, errors.New("error!")).Once()

	body, err := s.client.openFragment(location, readResult)
	c.Check(body, gc.IsNil)
	c.Check(err, gc.ErrorMatches, "error!")

	// Expect non-200 is turned into an error.
	mockClient.On("Get", "http://cloud/location").Return(&http.Response{
		StatusCode: http.StatusTeapot,
		Status:     "error!",
		Body:       ioutil.NopCloser(nil),
	}, nil).Once()

	body, err = s.client.openFragment(location, readResult)
	c.Check(body, gc.IsNil)
	c.Check(err, gc.ErrorMatches, "fetching fragment: error!")

	// Seek failure (too little content). Expect error is returned.
	mockClient.On("Get", "http://cloud/location").Return(&http.Response{
		StatusCode: http.StatusOK,
		Body:       ioutil.NopCloser(strings.NewReader("abc")),
	}, nil).Once()

	body, err = s.client.openFragment(location, readResult)
	c.Check(body, gc.IsNil)
	c.Check(err, gc.ErrorMatches, "seeking fragment: EOF")
}

func (s *ClientSuite) TestPut(c *gc.C) {
	content := strings.NewReader("foobar")

	mockClient := &mockHttpClient{}
	mockClient.On("Do", mock.MatchedBy(func(request *http.Request) bool {
		return request.Method == "HEAD" &&
			request.URL.Host == "default" &&
			request.URL.Path == "/a/journal"
	})).Return(newReadResponseFixture(), nil)

	body := new(bytes.Buffer)
	body.WriteString("some reason")

	// Return an error.
	mockClient.On("Do", mock.MatchedBy(func(request *http.Request) bool {
		return request.Method == "PUT" &&
			request.URL.Host == "redirected-server" &&
			request.URL.Path == "/a/journal" &&
			request.ContentLength == 6
	})).Return(&http.Response{
		Status:     "Internal Server Error",
		StatusCode: http.StatusInternalServerError,
		Request:    &http.Request{URL: newURL("http://default/a/journal")},
		Body:       ioutil.NopCloser(body),
		Header: http.Header{
			WriteHeadHeader: []string{"12341234"},
		},
	}, nil).Once()

	s.client.httpClient = mockClient
	res := s.client.Put(AppendArgs{Journal: "a/journal", Content: content})
	c.Check(res.Error, gc.ErrorMatches, `Internal Server Error \(some reason\)`)

	// WriteHead was parsed despite the failure.
	c.Check(res.WriteHead, gc.Equals, int64(12341234))

	// The broker will say that the write has failed, so we shouldn't hit the
	// counters for this journal. It has yet to be used at all, so it shouldn't
	// exist yet.
	c.Check(gazetteMap.Get("writers").(*expvar.Map).Get("a/journal"), gc.IsNil)

	// This time, return success.
	mockClient.On("Do", mock.MatchedBy(func(request *http.Request) bool {
		return request.Method == "PUT" &&
			// Cache is cleared, so "default" pops up again.
			request.URL.Host == "default" &&
			request.URL.Path == "/a/journal" &&
			request.ContentLength == 6
	})).Return(&http.Response{
		StatusCode: http.StatusNoContent, // Indicates success.
		Request:    &http.Request{URL: newURL("http://default/a/journal")},
		Body:       ioutil.NopCloser(nil),
		Header: http.Header{
			WriteHeadHeader: []string{"12341235"},
		},
	}, nil).Run(func(args mock.Arguments) {
		request := args[0].(*http.Request)
		c.Check(request.Body, gc.DeepEquals, ioutil.NopCloser(content))
	}).Once()

	res = s.client.Put(AppendArgs{Journal: "a/journal", Content: content})
	c.Check(res.Error, gc.IsNil)
	c.Check(res.WriteHead, gc.Equals, int64(12341235))
	mockClient.AssertExpectations(c)

	// Write success. Expect that the write stats were published to the
	// counters. ("content" is 6 bytes long, and is reflected in the bytes
	// counter.)
	writerMap := gazetteMap.Get("writers").(*expvar.Map).Get("a/journal").(*expvar.Map)
	c.Check(writerMap.Get("bytes").(*expvar.Int).String(), gc.Equals, "6")
	c.Check(writerMap.Get("head").(*expvar.Int).String(), gc.Equals, "12341235")
}

func (s *ClientSuite) TestReadResultParsingErrorCases(c *gc.C) {
	args := ReadArgs{Journal: "a/journal"}

	{ // Expect 416 is mapped into ErrNotYetAvailable.
		response := newReadResponseFixture()
		response.StatusCode = http.StatusRequestedRangeNotSatisfiable

		result, _ := s.client.parseReadResult(args, response)
		c.Check(result.Error, gc.Equals, ErrNotYetAvailable)
	}
	{ // Missing Content-Range.
		response := newReadResponseFixture()
		response.Header.Del("Content-Range")

		result, _ := s.client.parseReadResult(args, response)
		c.Check(result.Error, gc.ErrorMatches, "expected Content-Range header")
	}
	{ // Malformed Content-Range.
		response := newReadResponseFixture()
		response.Header.Set("Content-Range", "foobar")

		result, _ := s.client.parseReadResult(args, response)
		c.Check(result.Error, gc.ErrorMatches, "invalid Content-Range: foobar")
	}
	{ // Missing Write-Head.
		response := newReadResponseFixture()
		response.Header.Del(WriteHeadHeader)

		result, _ := s.client.parseReadResult(args, response)
		c.Check(result.Error, gc.ErrorMatches, "expected "+WriteHeadHeader+" header")
	}
	{ // Malformed Write-Head.
		response := newReadResponseFixture()
		response.Header.Set(WriteHeadHeader, "foobar")

		result, _ := s.client.parseReadResult(args, response)
		c.Check(result.Error, gc.ErrorMatches, "parsing "+WriteHeadHeader+": .*")
	}
	{ // Malformed fragment name.
		response := newReadResponseFixture()
		response.Header.Set(FragmentNameHeader, "foobar")

		result, _ := s.client.parseReadResult(args, response)
		c.Check(result.Error, gc.ErrorMatches, "parsing "+FragmentNameHeader+": .*")
	}
	{ // Malformed fragment location.
		response := newReadResponseFixture()
		response.Header.Set(FragmentLocationHeader, "@$%!@#@3")

		result, _ := s.client.parseReadResult(args, response)
		c.Check(result.Error, gc.ErrorMatches, "parsing "+FragmentLocationHeader+": .*")
	}
	{ // Non-206 response.
		response := newReadResponseFixture()
		response.StatusCode = http.StatusInternalServerError
		response.Status = "500 Internal Error"
		response.Body = ioutil.NopCloser(strings.NewReader("server error!"))

		result, _ := s.client.parseReadResult(args, response)
		c.Check(result.Error, gc.ErrorMatches, `500 Internal Error \(server error!\)`)
	}
}

func (s *ClientSuite) TestAppendResultParsingErrorCases(c *gc.C) {
	response := newReadResponseFixture()
	response.StatusCode = http.StatusNotFound
	response.Status = "404 Not Found"
	response.Body = ioutil.NopCloser(strings.NewReader("not found"))

	c.Check(s.client.parseAppendResponse(response).Error, gc.ErrorMatches,
		`404 Not Found \(not found\)`)
}

func (s *ClientSuite) TestBuildReadURL(c *gc.C) {
	args := ReadArgs{
		Journal:  "a/journal",
		Blocking: true,
		Deadline: time.Now().Add(10 * time.Millisecond),
	}
	url := s.client.buildReadURL(args)
	c.Check(strings.Contains(url.String(), "block=true"), gc.Equals, true)
	c.Check(strings.Contains(url.String(), "blockms="), gc.Equals, true)
	c.Check(strings.Contains(url.String(), "blockms=0"), gc.Equals, false)

	args = ReadArgs{Journal: "a/journal"}
	url = s.client.buildReadURL(args)
	c.Check(strings.Contains(url.String(), "block=false"), gc.Equals, true)
	c.Check(strings.Contains(url.String(), "blockms="), gc.Equals, false)
}

// Regression test for issue #890.
func (s *ClientSuite) TestDialerIsNonNil(c *gc.C) {
	// What we really want to test is that TCP keep-alive is set. There isn't a
	// great way to test this, as Dail is a closure over net.Dialer. At least
	// satisfy ourselves that a non-default dailer is used (a requirement for
	// automatic setting of TCP keep-alive).
	client, _ := NewClient("http://default")
	c.Check(client.httpClient.(*http.Client).Transport.(*http.Transport).Dial, gc.NotNil)
}

func newURL(s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}

var _ = gc.Suite(&ClientSuite{})