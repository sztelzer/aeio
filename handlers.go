package aeio

type Handler func(*Resource)

func HandleCreate(r *Resource) {
	if len(r.Errors) > 0 {
		Forbid(r)
		return
	}

	r.Create(true)
	if len(r.Errors) > 0 {
		Forbid(r)
		return
	}

	Allow(r)
	return
}

func HandleRead(r *Resource) {
	if len(r.Errors) > 0 {
		Forbid(r)
		return
	}

	r.Read()
	if len(r.Errors) > 0 {
		Forbid(r)
		return
	}

	Allow(r)
	return
}

func HandlePatch(r *Resource) {
	if len(r.Errors) > 0 {
		Forbid(r)
		return
	}

	r.Patch()
	if len(r.Errors) > 0 {
		Forbid(r)
		return
	}

	Allow(r)
	return
}

func HandleReadAll(r *Resource) {
	if len(r.Errors) > 0 {
		Forbid(r)
		return
	}

	r.ReadAll()
	if len(r.Errors) > 0 {
		Forbid(r)
		return
	}

	Allow(r)
	return
}

func HandleReadAny(r *Resource) {
	if len(r.Errors) > 0 {
		Forbid(r)
		return
	}

	r.ReadAny()
	if len(r.Errors) > 0 {
		Forbid(r)
		return
	}

	Allow(r)
	return
}

func HandleDelete(r *Resource) {
	if len(r.Errors) > 0 {
		Forbid(r)
		return
	}

	r.Delete()
	if len(r.Errors) > 0 {
		Forbid(r)
		return
	}

	Allow(r)
	return
}
