package main

import (
	"io"
	"os"
)

// Cikti: program çıktısının gittiği yer. Normalde ekran (stdout),
// wasm/tarayıcıda bir tampona yönlendirilip sayfaya basılır.
var Cikti io.Writer = os.Stdout
