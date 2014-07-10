package postit

import (
	"database/sql"
	"fmt"
	"github.com/ant0ine/go-json-rest/rest"
	"github.com/coopernurse/gorp"
	_ "github.com/lib/pq"
	"log"
	"net/http"
	"strconv"
	"time"
)

type Api struct {
	PageSize int
	dbMap    *gorp.DbMap
}

func (api *Api) InitDB(dbURL string) error {
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Error opening connection to %s: %v", dbURL, err)
		return err
	}
	api.dbMap = &gorp.DbMap{Db: db, Dialect: gorp.PostgresDialect{}}
	t1 := api.dbMap.AddTableWithName(Postit{}, "postits").SetKeys(true, "Id")
	t1.ColMap("Title").SetMaxSize(512)
	t1.ColMap("Category").SetMaxSize(256)
	err = api.dbMap.CreateTablesIfNotExists()
	if err != nil {
		log.Fatalf("Create tables failed with error: %v", err)
		return err
	}
	return nil
}

type Postit struct {
	Id         int64     `db:"post_id"`
	Title      string    `db:"title"`
	Body       string    `db:"body"`
	Created_on time.Time `db:"created_on"`
	Category   string    `db:"category"`
}

func (api *Api) GetAll(w rest.ResponseWriter, req *rest.Request) {
	// check if the query is limited to a category
	var postits []Postit
	var err error
	values := req.URL.Query()
	// check for pagination
	var pageStr, queryStr string
	_, ok := values["page"]
	if ok {
		pageNum, err := strconv.Atoi(values.Get("page"))
		if err != nil {
			log.Fatalf("invalid page number provided: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		pageStr = fmt.Sprintf("LIMIT %d OFFSET %d", api.PageSize, api.PageSize*pageNum)
	}
	_, ok = values["category"]
	if ok {
		queryStr = fmt.Sprintf("SELECT * FROM postits WHERE category=$1 ORDER BY created_on DESC %s", pageStr)
		_, err = api.dbMap.Select(&postits, queryStr, values.Get("category"))
	} else {
		queryStr = fmt.Sprintf("SELECT * FROM postits ORDER BY created_on DESC %s", pageStr)
		_, err = api.dbMap.Select(&postits, queryStr)
	}
	if err != nil {
		log.Fatalf("error retrieving records from database: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteJson(&postits)
}

func (api *Api) Get(w rest.ResponseWriter, req *rest.Request) {
	id := req.PathParam("id")
	p := Postit{}
	err := api.dbMap.SelectOne(&p, "SELECT * FROM postits WHERE post_id=$1", id)
	if err != nil {
		rest.NotFound(w, req)
		return
	}
	w.WriteJson(&p)
}

func (api *Api) Post(w rest.ResponseWriter, req *rest.Request) {
	p := Postit{}
	err := req.DecodeJsonPayload(&p)
	if err != nil {
		log.Fatalf("error decoding json payload: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if p.Category == "" || p.Title == "" || p.Body == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// create the database record
	p.Created_on = time.Now()
	err = api.dbMap.Insert(&p)
	if err != nil {
		log.Fatalf("Failed to create post record: %#v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	w.WriteJson(p)
}

func (api *Api) Put(w rest.ResponseWriter, req *rest.Request) {
	id := req.PathParam("id")
	p := Postit{}
	err := req.DecodeJsonPayload(&p)
	if err != nil {
		log.Fatalf("error decoding json payload: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	_, err = api.dbMap.Exec("UPDATE postits SET title=$1, body=$2, category=$3 WHERE post_id=$4", p.Title, p.Body, p.Category, id)
	if err != nil {
		log.Fatalf("error updating record: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (api *Api) Delete(w rest.ResponseWriter, req *rest.Request) {
	id, err := strconv.Atoi(req.PathParam("id"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	_, err = api.dbMap.Exec("DELETE FROM postits WHERE post_id=$1", id)
	if err != nil {
		log.Fatalf("error deleting record with id=%d: %v", id, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
