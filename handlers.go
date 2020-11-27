package aeio

type Handler func(*Resource) error

func HandlePut(r *Resource) error {
	return r.Put()
}

func HandleGet(r *Resource) error {
	return r.Get()
}

func HandleList(r *Resource) error {
	return r.GetMany()
}

func HandleListAny(r *Resource) error {
	return r.GetAny()
}

func HandleDelete(r *Resource) error {
	return r.Delete()
}
