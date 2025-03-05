package db

type Node struct {
	ID   string `bson:"_id"`
	Name string `bson:"name"`
}
