package svc

import (
	"reflect"
	"testing"
)

func TestTokenizeQuotesAndEmptyArg(t *testing.T) {
	got, err := tokenize(`{bin}/postgres -D {data} -p {port} -k ""`)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"{bin}/postgres", "-D", "{data}", "-p", "{port}", "-k", ""}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("tokenize = %#v, want %#v", got, want)
	}
}

func TestExpandSubstitutesAfterTokenizing(t *testing.T) {
	// A value containing a space must not split into two args.
	v := vars{
		bin: `C:\Program Files\pg\bin`, data: `C:\my data\pgdata`,
		port: 5432, user: "hey_x", password: "s3cret", pwfile: `C:\tmp\pw`,
	}
	got, err := expand("{bin}/initdb -D {data} -U {user} --pwfile {pwfile}", v)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		`C:\Program Files\pg\bin/initdb`,
		"-D", `C:\my data\pgdata`,
		"-U", "hey_x",
		"--pwfile", `C:\tmp\pw`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("expand = %#v, want %#v", got, want)
	}
}

func TestTokenizeUnterminatedQuote(t *testing.T) {
	if _, err := tokenize(`foo "bar`); err == nil {
		t.Error("expected error on unterminated quote")
	}
}
