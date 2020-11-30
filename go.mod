module github.com/sztelzer/aeio

go 1.14

require (
	cloud.google.com/go/datastore v1.3.0
	firebase.google.com/go/v4 v4.1.0
	github.com/sztelzer/structpatch v0.0.2
	google.golang.org/api v0.35.0
)

replace github.com/sztelzer/structpatch v0.0.2 => ../structpatch
