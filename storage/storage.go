package storage

type Key string

type StorageReader interface {
	Read(Key) Resource
}


