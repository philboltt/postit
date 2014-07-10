package postit

import (
	"fmt"
	"github.com/ant0ine/go-json-rest/rest"
	"github.com/ant0ine/go-json-rest/rest/test"
	"testing"
)

const test_config = "test_config.ini"

func Test_Postit(t *testing.T) {
	api := Api{}
	err := api.readConfig(test_config)
	if err != nil {
		t.Fatalf("failed to read %s: %v", test_config, err)
	}
	// Initialize database connection
	err = api.initDB()
	if err != nil {
		t.Fatalf("failed to open database connection: %v", err)
	}
	defer api.dbMap.Db.Close()
	// clear any existing data
	err = api.dbMap.TruncateTables()
	if err != nil {
		t.Fatalf("error truncating tables: %v", err)
	}
	// set up the REST routes
	rh := &rest.ResourceHandler{}
	rh.SetRoutes(
		&rest.Route{"GET", "/posts", api.GetAll},
		&rest.Route{"GET", "/posts/:id", api.Get},
		&rest.Route{"POST", "/posts", api.Post},
		&rest.Route{"PUT", "/posts/:id", api.Put},
		&rest.Route{"DELETE", "/posts/:id", api.Delete},
	)
	// this should return BadRequest
	p := Postit{}
	req := test.MakeSimpleRequest("POST", "http://localhost:8080/posts", p)
	recordered := test.RunRequest(t, rh, req)
	recordered.CodeIs(400)
	// this should return 200 Ok, and the id of the newly created record
	p = Postit{
		Title:    "This is test #1",
		Category: "code",
		Body:     "This is the body data",
	}
	req = test.MakeSimpleRequest("POST", "http://localhost:8080/posts", p)
	recordered = test.RunRequest(t, rh, req)
	recordered.CodeIs(201)
	result := Postit{}
	err = recordered.DecodeJsonPayload(&result)
	if err != nil {
		t.Fatalf("error decoding returning payload: %v", err)
	}
	if result.Id == 0 {
		t.Fatalf("expected non-zero returning id from post, got %d", result.Id)
	}
	// retrieve the newly created record and compare values
	url := fmt.Sprintf("http://localhost:8080/posts/%d", result.Id)
	req = test.MakeSimpleRequest("GET", url, nil)
	recordered = test.RunRequest(t, rh, req)
	recordered.CodeIs(200)
	p2 := Postit{}
	recordered.DecodeJsonPayload(&p2)
	// note: timestamps won't match because PG truncates the fractional component of the seconds
	// and the POST action doesn't retrieve the record fresh from the DB
	if result.Id != p2.Id || result.Category != p2.Category || result.Title != p2.Title || result.Body != p2.Body {
		t.Fatalf("created and retrieved records don't match: %v vs %v", result, p2)
	}
	// update a record
	p2.Body = "This is an updated body text"
	url = fmt.Sprintf("http://localhost:8080/posts/%d", p2.Id)
	req = test.MakeSimpleRequest("PUT", url, p2)
	recordered = test.RunRequest(t, rh, req)
	recordered.CodeIs(202)
	// check the updated record
	req = test.MakeSimpleRequest("GET", url, nil)
	recordered = test.RunRequest(t, rh, req)
	recordered.CodeIs(200)
	p3 := Postit{}
	recordered.DecodeJsonPayload(&p3)
	if p2.Body != p3.Body {
		t.Fatalf("failed to update record body: %v", p3)
	}
	// delete a record
	url = fmt.Sprintf("http://localhost:8080/posts/%d", p3.Id)
	req = test.MakeSimpleRequest("DELETE", url, nil)
	recordered = test.RunRequest(t, rh, req)
	recordered.CodeIs(200)
	// confirm record is deleted
	req = test.MakeSimpleRequest("GET", url, nil)
	recordered = test.RunRequest(t, rh, req)
	recordered.CodeIs(404)
	// create a bunch of new records
	for i := 0; i < 50; i++ {
		var title, category string
		title = fmt.Sprintf("Record #%d", i)
		if i%2 == 0 {
			category = "code"
		} else {
			category = "rant"
		}
		p = Postit{
			Title:    title,
			Category: category,
			Body:     "This is the body data",
		}
		req = test.MakeSimpleRequest("POST", "http://localhost:8080/posts", p)
		recordered = test.RunRequest(t, rh, req)
		recordered.CodeIs(201)
	}
	// retrieve all records
	url = "http://localhost:8080/posts"
	req = test.MakeSimpleRequest("GET", url, nil)
	recordered = test.RunRequest(t, rh, req)
	recordered.CodeIs(200)
	var results []Postit
	recordered.DecodeJsonPayload(&results)
	if len(results) != 50 {
		t.Fatalf("Expected 50 records, got %d", len(results))
	}
	// retrieve all records in a given category
	url = "http://localhost:8080/posts?category=code"
	req = test.MakeSimpleRequest("GET", url, nil)
	recordered = test.RunRequest(t, rh, req)
	recordered.CodeIs(200)
	recordered.DecodeJsonPayload(&results)
	if len(results) != 25 {
		t.Fatalf("Expected 25 records, got %d", len(results))
	}
	url = "http://localhost:8080/posts?category=rant"
	req = test.MakeSimpleRequest("GET", url, nil)
	recordered = test.RunRequest(t, rh, req)
	recordered.CodeIs(200)
	recordered.DecodeJsonPayload(&results)
	if len(results) != 25 {
		t.Fatalf("Expected 25 records, got %d", len(results))
	}
	// retrieve a page of records
	url = "http://localhost:8080/posts?page=0"
	req = test.MakeSimpleRequest("GET", url, nil)
	recordered = test.RunRequest(t, rh, req)
	recordered.CodeIs(200)
	recordered.DecodeJsonPayload(&results)
	if len(results) != 10 {
		t.Fatalf("Expected 10 records, got %d", len(results))
	}
	// retrieve a page of records in a given category
	url = "http://localhost:8080/posts?page=0&category=rant"
	req = test.MakeSimpleRequest("GET", url, nil)
	recordered = test.RunRequest(t, rh, req)
	recordered.CodeIs(200)
	recordered.DecodeJsonPayload(&results)
	if len(results) != 10 {
		t.Fatalf("Expected 10 records, got %d", len(results))
	}
	for i := 0; i < 10; i++ {
		if results[i].Category != "rant" {
			t.Fatalf("CAtegory should be 'rant', but got %s", results[i].Category)
		}
	}
}
