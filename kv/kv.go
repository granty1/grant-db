package kv

// Storage defines the interface for storage
type Storage interface {
	store()
}

type tempStorage struct {}

func (ts *tempStorage) store(){}

func NewStorage() Storage {
	return &tempStorage{}
}