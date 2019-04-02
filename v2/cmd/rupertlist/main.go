package main

import (
	"context"
	"fmt"

	"cloud.google.com/go/storage"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

func main() {
	log.Info("Hello")

	ctx := context.Background()

	// Creates a client.
	client, err := storage.NewClient(ctx, option.WithCredentialsFile("/home/rupche/gcs-read-only.json"))
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	//listBuckets(ctx, client)
	listObjects(ctx, client)
}

func listBuckets(ctx context.Context, client *storage.Client) {
	it := client.Buckets(ctx, "liveramp-eng-ps")
	for {
		battrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		log.Info(battrs.Name)
	}
}

func listObjects(ctx context.Context, client *storage.Client) {
	bucket := client.Bucket("com-liveramp-ps-us-test")
	q := &storage.Query{
		Prefix: "rupert/part-000/",
		//Delimiter: "/",
	}
	fmt.Printf("%+v\n", q)
	it := bucket.Objects(ctx, q)

	log.Info("Got objects")
	for obj, err := it.Next(); err == nil; obj, err = it.Next() {
		fmt.Printf("%s, %s, %s\n", obj.Name, obj.ContentType, obj.Prefix)
		//fmt.Printf("%#v\n", obj)
		//if obj.Name[len(obj.Name)-1] == '/' {
		//	fmt.Printf("that was a directory!")
		//}
	}
}

func foo() {
	creds, err := google.FindDefaultCredentials(context.Background(), storage.ScopeFullControl)
	if err != nil {
		panic(err)
	}

	conf, err := google.JWTConfigFromJSON(creds.JSON, storage.ScopeFullControl)
	if err != nil {
		panic(err)
	}

	client, err := storage.NewClient(context.Background(), option.WithTokenSource(conf.TokenSource(context.Background())))
	if err != nil {
		panic(err)
	}

	fmt.Printf("%v\n", client)
}
