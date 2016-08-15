package aeio

// import (
// 	"heartbend/resio"
// )

//HANDLERS SHOULD WRITE TO BUFFER, BUT MUST OUTPUT TO WRITER (INCLUDING ERRORS FROM RESIO)
//RESIO DON'T CARE FOR ERRORS BACK, BUT IT CAN USE THE RESOURCE TO LOG OR WHATEVER.
//THERE IS THE POSSIBILITY THAT HANDLERS ARE CALLED WITH ERRORS ALREADY SET. CONSIDER THEM!

func Create(r *Resource) {
	if len(r.Errors) > 0 {
		Forbid(r)
		return
	}

	r.Create(false)
	if len(r.Errors) > 0 {
		Forbid(r)
		return
	}

	Allow(r)
	return
}

func Read(r *Resource) {
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

//
// func Patch(r *Resource) {
// 	if len(r.Errors) > 0 {
// 		Forbid(r)
// 		return
// 	}
//
// 	r.Patch()
// 	if len(r.Errors) > 0 {
// 		Forbid(r)
// 		return
// 	}
//
// 	// j, _ := json.Marshal(r)
// 	// r.Access.Writer.Write(j)
// 	Allow(r)
// 	return
// }
//
func ReadAll(r *Resource) {
	if len(r.Errors) > 0 {
		Forbid(r)
		return
	}

	r.ReadAll()
	if len(r.Errors) > 0 {
		Forbid(r)
		return
	}

	// j, _ := json.Marshal(r)
	// r.Access.Writer.Write(j)
	Allow(r)
	return
}

//
// func HandleReadAny(r *resio.Resource) {
// 	if len(r.Errors) > 0 {
// 		Forbid(r)
// 		return
// 	}
//
// 	r.ReadAny()
// 	if len(r.Errors) > 0 {
// 		Forbid(r)
// 		return
// 	}
//
// 	// j, _ := json.Marshal(r)
// 	// r.Access.Writer.Write(j)
// 	Allow(r)
// 	return
// }
//
// func HandleDelete(r *resio.Resource) {
// 	if len(r.Errors) > 0 {
// 		Forbid(r)
// 		return
// 	}
//
// 	r.Delete()
// 	if len(r.Errors) > 0 {
// 		Forbid(r)
// 		return
// 	}
//
// 	Allow(r)
// 	return
// }
//
// func HandleUserToken(r *resio.Resource) {
// 	if len(r.Errors) > 0 {
// 		Forbid(r)
// 		return
// 	}
//
// 	r.ReadNewToken()
// 	if len(r.Errors) > 0 {
// 		Forbid(r)
// 		return
// 	}
//
// 	// j, _ := json.Marshal(r)
// 	// r.Access.Writer.Write(j)
// 	Allow(r)
// 	return
// }
//
// func HandleOpenFeedbacks(r *resio.Resource) {
// 	if len(r.Errors) > 0 {
// 		Forbid(r)
// 		return
// 	}
//
// 	resio.GetOpenFeedbacks(r)
// 	if len(r.Errors) > 0 {
// 		Forbid(r)
// 		return
// 	}
//
// 	Allow(r)
// 	return
//
// }
//
// func HandleUserResetPass(r *resio.Resource) {
// 	if len(r.Errors) > 0 {
// 		Forbid(r)
// 		return
// 	}
//
// 	r.SendResetUserPass()
// 	r.Object = nil
// 	if len(r.Errors) > 0 {
// 		Forbid(r)
// 		return
// 	}
//
// 	// j, _ := json.Marshal(r)
// 	// r.Access.Writer.Write(j)
// 	Allow(r)
// 	return
// }
//
// func HandleTestResetToken(r *resio.Resource) {
// 	if len(r.Errors) > 0 {
// 		Forbid(r)
// 		return
// 	}
//
// 	Allow(r)
// 	return
// }
//
// func HandlePatchPassword(r *resio.Resource) {
// 	if len(r.Errors) > 0 {
// 		Forbid(r)
// 		return
// 	}
// 	r.PatchPassword()
// 	if len(r.Errors) > 0 {
// 		Forbid(r)
// 		return
// 	}
//
// 	// j, _ := json.Marshal(r)
// 	// r.Access.Writer.Write(j)
// 	Allow(r)
// 	return
// }
//
// func HandleTrainFullModel(r *resio.Resource) {
// 	userKey := r.Key.Parent()
// 	err := resio.TrainFullModel(r, userKey)
// 	if err != nil {
// 		r.E("training_full_model", err)
// 	}
// }
//
// func HandleCheckTrainingModel(r *resio.Resource) {
// 	userKey := r.Key.Parent()
// 	err := resio.ModelRunning(r, userKey)
// 	if err != nil {
// 		r.E("model_status", err)
// 	}
// }
