module github.com/sztelzer/aeio

go 1.14

replace github.com/sztelzer/structpatch v0.0.2 => ../structpatch

require (
	cloud.google.com/go/datastore v1.4.0
	firebase.google.com/go/v4 v4.2.0
	github.com/sztelzer/structpatch v0.0.2
	google.golang.org/api v0.39.0
)
