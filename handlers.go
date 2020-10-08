package aeio

type Handler func(*Resource) error

func HandleCreate(r *Resource) error {
	return r.Create()
}

func HandleRead(r *Resource) error {
	return r.Read()
}

func HandlePatch(r *Resource) error {
	return r.Patch()
}

func HandleReadAll(r *Resource) error {
	return r.ReadAll()
}

func HandleReadAny(r *Resource) error {
	return r.ReadAny()
}

func HandleDelete(r *Resource) error {
	return r.Delete()
}
