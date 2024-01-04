package database

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/dapr/components-contrib/bindings"
	"github.com/dapr/components-contrib/bindings/postgres"
	"github.com/dapr/components-contrib/metadata"
	"github.com/dapr/kit/logger"
)

const (
	testInsert   = "INSERT INTO sagastate (key, value) values ('%s', '%s')"
	testDelete   = "DELETE FROM sagastate WHERE key = '%s';"
	testSelect   = "SELECT key, value FROM sagastate;"
	rowsAffected = "rows-affected"
)

type retdata struct {
	key   string
	value string
}

func main() {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		log.Fatalln("DATABASE_URL environment variable needs to be set")
	}

	// live DB test
	b := postgres.NewPostgres(logger.NewLogger("test")).(*postgres.Postgres)
	m := bindings.Metadata{Base: metadata.Base{Properties: map[string]string{"connectionString": url}}}
	if err := b.Init(context.Background(), m); err != nil {
		log.Fatal(err)
	}

	req := &bindings.InvokeRequest{
		Operation: "exec",
		Metadata:  map[string]string{},
	}

	params := "{}"
	params = base64.StdEncoding.EncodeToString([]byte(params))
	t := time.Now().UTC()
	s1 := t.String()

	log_m := `{"app_id":` + "test1" + ","
	log_m += `"service":` + "testservice" + ","
	log_m += `"token":` + "abcd123" + ","
	log_m += `"callback_service":` + "dummy" + ","
	log_m += `"params":` + params + ","
	log_m += `"event": true` + ","
	log_m += `"timeout":` + strconv.Itoa(10) + ","
	log_m += `"logtime":` + s1 + "}"
	key := "mykey"
	data := base64.StdEncoding.EncodeToString([]byte(log_m))
	req.Metadata["sql"] = fmt.Sprintf(testInsert, key, data)
	ctx := context.TODO()
	// Insert
	res, err := b.Invoke(ctx, req)
	if err != nil {
		log.Fatalln("Error on insert", err)
	}
	log.Printf("Insert res = %v\n", res)
	if res.Metadata[rowsAffected] != "1" {
		log.Fatalln("Error on insert row count")
	}

	// Query
	req.Operation = "query"
	req.Metadata["sql"] = testSelect
	res, err = b.Invoke(ctx, req)
	if err != nil {
		log.Fatalln("Error on Query", err)
	}
	log.Printf("Query res = %v\n", res)
	// Lets try a simple unmarshall

	ret := getRows(res.Data)
	log.Printf("Query res unmarshalled  = %v\n", ret)
	// Delete
	req.Operation = "exec"
	req.Metadata["sql"] = fmt.Sprintf(testDelete, key)
	res, err = b.Invoke(ctx, req)
	if err != nil {
		log.Fatalln("Error on delete", err)
	}
	log.Printf("Insert res = %v\n", res)
	if res.Metadata[rowsAffected] != "1" {
		log.Fatalln("Error on delete row count")
	}

	req.Operation = "close"
	req.Metadata = nil
	req.Data = nil
	_, err = b.Invoke(ctx, req)
	if err != nil {
		log.Fatalln("Error on close", err)
	}

	err = b.Close()
	if err != nil {
		log.Fatalln("Error on binding close", err)
	}

}

// [[\"one\",\"two\"],[\"mykey\",\"eyJhcHBfaWQiOnRlc3QxLCJzZXJ2aWNlIjp0ZXN0c2VydmljZSwidG9rZW4iOmFiY2QxMjMsImNhbGxiYWNrX3NlcnZpY2UiOmR1bW15LCJwYXJhbXMiOmUzMD0sImV2ZW50IjogdHJ1ZSwidGltZW91dCI6MTAsImxvZ3RpbWUiOjIwMjQtMDEtMDMgMTU6MTI6NDEuOTQ4MDIgKzAwMDAgVVRDfQ==\"]]"
func getRows(input []byte) []retdata {
	var ret []retdata
	re := regexp.MustCompile(`(\w+)`)
	split1 := re.FindAllStringSubmatch(string(input), -1)
	ret = make([]retdata, len(split1)/2)
	var index = 0
	for i, v := range split1 {
		if ((i + 1) % 2) == 0 {
			ret[index].value = v[0]
			index = index + 1
		} else {
			ret[index].key = v[0]
		}
	}
	return ret
}
