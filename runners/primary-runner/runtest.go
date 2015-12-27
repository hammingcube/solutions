package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
)

const (
	PENDING = "pending"
	SUCCESS = "success"
	ERROR   = "error"
	FAILURE = "failure"
)

func runProg(cmd *exec.Cmd) (io.WriteCloser, io.ReadCloser, error) {
	w, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}
	err = cmd.Start()
	if err != nil {
		return nil, nil, err
	}
	return w, stdout, nil
}

var inputLog, w1Log, w2Log bytes.Buffer

func runIt(r io.Reader, prog1 *exec.Cmd, prog2 *exec.Cmd) (io.ReadCloser, io.ReadCloser) {

	iw := bufio.NewWriter(&inputLog)
	w1, r1, err := runProg(prog1)
	if err != nil {
		log.Fatal(err)
	}
	w2, r2, err := runProg(prog2)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		defer w1.Close()
		defer w2.Close()
		defer iw.Flush()
		mw := io.MultiWriter(w1, w2, iw)
		io.Copy(mw, r)
	}()

	return r1, r2
}

func main() {
	genBinary, prog1Binary, prog2Binary := os.Args[1], os.Args[2], os.Args[3]

	generator := exec.Command(genBinary)
	r, err := generator.StdoutPipe()
	if err != nil {
		fmt.Println(err)
	}

	prog1 := exec.Command(prog1Binary)
	prog2 := exec.Command(prog2Binary)

	r1, r2 := runIt(r, prog1, prog2)

	generator.Run()

	areDifferent := diff2(r1, r2)
	status := PENDING
	if areDifferent {
		status = FAILURE
	} else {
		status = SUCCESS
	}
	statusJson, err := json.Marshal(map[string]string{"status": status})
	if err != nil {
		log.Fatal(err)
	}
	err = prog1.Wait()
	err1 := prog2.Wait()
	if err != nil || err1 != nil {
		fmt.Println(err, err1)
	}
	ioutil.WriteFile("input.txt", inputLog.Bytes(), 0644)
	ioutil.WriteFile("out1.txt", w1Log.Bytes(), 0644)
	ioutil.WriteFile("out2.txt", w2Log.Bytes(), 0644)
	ioutil.WriteFile("status.json", statusJson, 0644)

	//fmt.Printf("inputLog: %s\n", &inputLog)
	//fmt.Printf("w1Log: %s\n", &w1Log)
	//fmt.Printf("w2Log: %s\n", &w2Log)
}

func diff2(r1, r2 io.Reader) bool {
	diff := false
	iw1 := bufio.NewWriter(&w1Log)
	iw2 := bufio.NewWriter(&w2Log)

	defer iw1.Flush()
	defer iw2.Flush()

	tr1 := io.TeeReader(r1, iw1)
	tr2 := io.TeeReader(r2, iw2)

	scanner1 := bufio.NewScanner(tr1)
	scanner2 := bufio.NewScanner(tr2)
	for {
		n1 := scanner1.Scan()
		n2 := scanner2.Scan()
		err1 := scanner1.Err()
		err2 := scanner1.Err()
		//fmt.Println(n1, n2, err1, err2)
		if n1 != n2 || n1 == false || n2 == false {
			break
		}
		if err1 != nil || err2 != nil {
			break
		}
		line1 := scanner1.Text()
		line2 := scanner2.Text()
		//fmt.Printf("So far: %s, %s\n", line1, line2)
		if line1 != line2 {
			diff = true
			fmt.Printf("Mismatch:\n->%s\n=>%s\n", line1, line2)
		}
	}
	return diff
}
