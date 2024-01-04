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
	req.Metadata["sql"] = stateSelect
	res, err := the_db.Invoke(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("Error on Query %s", err)
	}
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
	//log.Printf("DB:Store Data = %s\n", base64.URLEncoding.EncodeToString([]byte(value)))
	req.Operation = operationExec
	req.Metadata[sql] = fmt.Sprintf(stateInsert, key, base64.URLEncoding.EncodeToString([]byte(value)))
	res, err := the_db.Invoke(ctx, req)
	if err != nil {
		return fmt.Errorf("Error on insert for key %s %s", key, err)
	}
	if res.Metadata[theRowsAffected] != "1" {
		return fmt.Errorf("error on insert row count wrong for key %s", key)
	}

	return err
}

// [[\"one\",\"two\"],[\"mykey\",\"eyJhcHBfaWQiOnRlc3QxLCJzZXJ2aWNlIjp0ZXN0c2VydmljZSwidG9rZW4iOmFiY2QxMjMsImNhbGxiYWNrX3NlcnZpY2UiOmR1bW15LCJwYXJhbXMiOmUzMD0sImV2ZW50IjogdHJ1ZSwidGltZW91dCI6MTAsImxvZ3RpbWUiOjIwMjQtMDEtMDMgMTU6MTI6NDEuOTQ4MDIgKzAwMDAgVVRDfQ==\"]]"
func getTheRows(input []byte) []StateRecord {
	var ret []StateRecord
	re := regexp.MustCompile(`(\w+)`)
	split1 := re.FindAllStringSubmatch(string(input), -1)
	//log.Printf("split1  = %v\n", split1)
	//log.Printf("len split1  = %v\n", len(split1))
	size := len(split1)
	if size > 0 {
		l := size / 2
		if l == 0 {
			l = 1
		}
		ret = make([]StateRecord, l)
		var index = 0
		for i, v := range split1 {
			if ((i + 1) % 2) == 0 {
				ret[index].Value = v[0] + "==" // Ensure base64 data is valid
				index = index + 1
			} else {
				ret[index].Key = v[0]
			}
		}
	}
	return ret
}
