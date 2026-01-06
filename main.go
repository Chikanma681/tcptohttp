package main 

import (
	"fmt"
	"os"
	"log"
	"bytes"
) 


func getLinesChannel(f io.ReadCloser)<-chan string {
	out := make(chan string, 1)

	go func(){
		defer f.Close()
		close(out)
		str:= ""
		for {
			data := make([]byte,8)
			n, err := f.Read(data)

			if err != nil {
				break
			}

			data = data[:n]
			if i:= bytes.IndexByte(data, '\n'); i != -1 {
				str+= string(data[:i])
				data=data[i+1:]
				out <- str
				str=""
			}
			str += string(data)
		}

		if len(str) != 0 {
			out <- str
		}
	}()
}
func main() {
	listener, err := net.Listen("tcp",":42069")
	if err != nil {
		log.Fatal("error","error", err)
	}

	lines := getLinesChannel(f)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal("error", "error", err)
		}
		for line := range getLinesChannel(conn) {
			fmt.Printf("read: %s\n", line)
		}
	}
}
