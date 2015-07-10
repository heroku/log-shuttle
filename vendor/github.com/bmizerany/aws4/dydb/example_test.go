package dydb_test

import (
	"fmt"
	"github.com/bmizerany/aws4/dydb"
	"log"
)

func init() {
	log.SetFlags(0)
}

func Example_createAndListTables() {
	var db dydb.DB

	type AttributeDefinition struct {
		AttributeName string
		AttributeType string
	}

	type KeySchema struct {
		AttributeName string
		KeyType       string
	}

	var posts struct {
		TableName             string
		AttributeDefinitions  []AttributeDefinition
		KeySchema             []KeySchema
		ProvisionedThroughput struct {
			ReadCapacityUnits  int
			WriteCapacityUnits int
		}
	}

	posts.TableName = "Posts"
	posts.AttributeDefinitions = []AttributeDefinition{{"Slug", "S"}}
	posts.KeySchema = []KeySchema{{"Slug", "HASH"}}
	posts.ProvisionedThroughput.ReadCapacityUnits = 4
	posts.ProvisionedThroughput.WriteCapacityUnits = 4

	if err := db.Exec("CreateTable", posts); err != nil {
		if !dydb.IsException(err, "ResourceInUseException") {
			log.Fatal(err)
		}
	}

	var resp struct{ TableNames []string }
	if err := db.Query("ListTables", nil).Decode(&resp); err != nil {
		log.Fatal(err)
	}

	// Output:
	// ["Posts"]
	fmt.Printf("%q", resp.TableNames)
}
