package aeio

type Handler func(*Resource) error

func HandleCreate(r *Resource) error {
	return r.Create()
}

func HandleUpdate(r *Resource) error {
	return r.Update()
}

func HandleGet(r *Resource) error {
	return r.Get()
}

func HandleGetList(r *Resource) error {
	return r.GetMany()
}

func HandleGetAny(r *Resource) error {
	return r.GetAny()
}

func HandleDelete(r *Resource) error {
	return r.Delete()
}
