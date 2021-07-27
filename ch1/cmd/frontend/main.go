package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	redis "github.com/go-redis/redis/v8"
)

const journalKey = "journal-key"

var (
	rdb *redis.Client

	redisAddr = ":6379"
)

type Data []datum

func (d Data) MarshalBinary() ([]byte, error) {
	return json.Marshal(d)
}

type datum interface{}

func init() {
	if redisEnvAddr := os.Getenv("REDIS_ADDR"); redisEnvAddr != "" {
		redisAddr = redisEnvAddr
	}
}

func apiHandler(resp http.ResponseWriter, req *http.Request) {
	log.Println("in apiHandler")
	ctx := req.Context()
	val := rdb.Get(ctx, journalKey)
	buf, err := val.Bytes()
	if err != nil {
		log.Printf("could not get journal: %v", err)
	}
	var existingData Data
	if buf != nil {
		if err := json.Unmarshal(buf, &existingData); err != nil {
			log.Printf("could not unmarshal saved data: %v", err)
			resp.WriteHeader(500)
			return
		}
	}

	switch req.Method {
	case "GET":
		resp.WriteHeader(200)
		resp.Write(buf)
		return
	case "POST":
		decoder := json.NewDecoder(req.Body)
		var body interface{}
		if err := decoder.Decode(&body); err != nil {
			log.Printf("could not decode request body: %v", err)
			resp.WriteHeader(500)
			return
		}
		existingData = append(existingData, body)

		res := rdb.Set(ctx, journalKey, existingData, 0)
		if _, err := res.Result(); err != nil {
			log.Printf("could not set the journal in redis: %v", err)
			resp.WriteHeader(500)
			return
		}

		resp.WriteHeader(200)
		encoder := json.NewEncoder(resp)
		if err := encoder.Encode(existingData); err != nil {
			log.Printf("could not encode json to response writer: %v", err)
			resp.WriteHeader(500)
			return
		}

	default:

	}
}

func main() {
	rdb = redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	http.HandleFunc("/api", apiHandler)
	log.Println(http.ListenAndServe(":8080", nil))
}
