package database

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"regexp"

	"github.com/dapr/components-contrib/bindings"
	"github.com/dapr/components-contrib/bindings/postgres"
)

const (
	stateInsert     = "INSERT INTO sagastate (key, value) values ('%s', '%s')"
	stateDelete     = "DELETE FROM sagastate WHERE key = '%s';"
	stateSelect     = "SELECT key, value FROM sagastate;"
	theRowsAffected = "rows-affected"
	operationExec   = "exec"
	sql             = "sql"
)

type StateRecord struct {
	Key   string
	Value string
}

var req = &bindings.InvokeRequest{
	Operation: operationExec,
	Metadata:  map[string]string{},
}

func GetStateRecords(ctx context.Context, the_db *postgres.Postgres) ([]StateRecord, error) {
	var err error

	req.Operation = "query"
	req.Metadata["sql"] = testSelect
	res, err := the_db.Invoke(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("Error on Query %s", err)
	}
	log.Printf("Query res = %v\n", res)
	ret := getTheRows(res.Data)
	return ret, err
}

func Delete(ctx context.Context, the_db *postgres.Postgres, key string) error {
	var err error

	log.Printf("DB:Delete Key = %s\n", key)

	req.Operation = operationExec
	req.Metadata[sql] = fmt.Sprintf(stateDelete, key)
	res, err := the_db.Invoke(ctx, req)
	if err != nil {
		return fmt.Errorf("Error on delete for key %s, err %s", key, err)
	}
	if res.Metadata[theRowsAffected] != "1" {
		return fmt.Errorf("Error on delete row count %s for key = %s", res.Metadata[theRowsAffected], key)
	}
	return err
}

func StoreState(ctx context.Context, the_db *postgres.Postgres, key string, value []byte) error {
	var err error

	log.Printf("DB:Store Key = %s\n", key)
	req.Operation = operationExec
	req.Metadata[sql] = fmt.Sprintf(stateInsert, key, base64.StdEncoding.EncodeToString([]byte(value)))
	res, err := the_db.Invoke(ctx, req)
	if err != nil {
		return fmt.Errorf("Error on insert for key %s", key, err)
	}

	log.Printf("Insert res = %v\n", res)
	if res.Metadata[theRowsAffected] != "1" {
		return fmt.Errorf("Error on insert row count wrong for key %s", key)
	}

	return err
}

// [[\"one\",\"two\"],[\"mykey\",\"eyJhcHBfaWQiOnRlc3QxLCJzZXJ2aWNlIjp0ZXN0c2VydmljZSwidG9rZW4iOmFiY2QxMjMsImNhbGxiYWNrX3NlcnZpY2UiOmR1bW15LCJwYXJhbXMiOmUzMD0sImV2ZW50IjogdHJ1ZSwidGltZW91dCI6MTAsImxvZ3RpbWUiOjIwMjQtMDEtMDMgMTU6MTI6NDEuOTQ4MDIgKzAwMDAgVVRDfQ==\"]]"
func getTheRows(input []byte) []StateRecord {
	var ret []StateRecord
	re := regexp.MustCompile(`(\w+)`)
	split1 := re.FindAllStringSubmatch(string(input), -1)
	ret = make([]StateRecord, len(split1)/2)
	var index = 0
	for i, v := range split1 {
		if ((i + 1) % 2) == 0 {
			ret[index].Value = v[0]
			index = index + 1
		} else {
			ret[index].Key = v[0]
		}
	}
	return ret
}
