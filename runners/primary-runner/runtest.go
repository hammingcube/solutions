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

// =====================

// Original Source: https://raw.githubusercontent.com/cespare/diff/master/diff.go

const chunkSize = 4096

// Readers compares the contents of two io.Readers.
// The return value of different is true if and only if there are no errors
// in reading r1 and r2 (io.EOF excluded) and r1 and r2 are
// byte-for-byte identical.
func DiffReaders(r1, r2 io.Reader) (different bool, err error) {
	buf1 := make([]byte, chunkSize)
	buf2 := make([]byte, chunkSize)
	for {
		short1 := false
		n1, err := io.ReadFull(r1, buf1)
		switch err {
		case io.EOF, io.ErrUnexpectedEOF:
			short1 = true
		case nil:
		default:
			return true, err
		}
		short2 := false
		n2, err := io.ReadFull(r2, buf2)
		switch err {
		case io.EOF, io.ErrUnexpectedEOF:
			short2 = true
		case nil:
		default:
			return true, err
		}
		if short1 != short2 || n1 != n2 {
			return true, nil
		}
		if !bytes.Equal(buf1[:n1], buf2[:n1]) {
			return true, nil
		}
		if short1 {
			return false, nil
		}
	}
}

// Files compares the contents of file1 and file2.
// Files first compares file length before looking at the contents.
func DiffFiles(file1, file2 string) (different bool, err error) {
	f1, err := os.Open(file1)
	if err != nil {
		return true, err
	}
	defer f1.Close()
	f2, err := os.Open(file2)
	if err != nil {
		return true, err
	}
	defer f2.Close()

	// Compare the size of the files.
	n1, err := f1.Seek(0, os.SEEK_END)
	if err != nil {
		return true, err
	}
	n2, err := f2.Seek(0, os.SEEK_END)
	if err != nil {
		return true, err
	}
	if n1 != n2 {
		return true, nil
	}
	if _, err := f1.Seek(0, os.SEEK_SET); err != nil {
		return true, err
	}
	if _, err := f2.Seek(0, os.SEEK_SET); err != nil {
		return true, err
	}

	// Otherwise compare the contents.
	return DiffReaders(f1, f2)
}

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

	status := PENDING
	if areDifferent(r1, r2) {
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
	log.Printf("%s\n", statusJson)
	log.Printf("%s\n", w1Log.Bytes())
	log.Printf("%s\n", w2Log.Bytes())
	ioutil.WriteFile("input.txt", inputLog.Bytes(), 0644)
	ioutil.WriteFile("out1.txt", w1Log.Bytes(), 0644)
	ioutil.WriteFile("out2.txt", w2Log.Bytes(), 0644)
	ioutil.WriteFile("status.json", statusJson, 0644)

	//fmt.Printf("inputLog: %s\n", &inputLog)
	//fmt.Printf("w1Log: %s\n", &w1Log)
	//fmt.Printf("w2Log: %s\n", &w2Log)
}

func areDifferent(r1, r2 io.Reader) bool {
	same := false
	iw1 := bufio.NewWriter(&w1Log)
	iw2 := bufio.NewWriter(&w2Log)

	defer iw1.Flush()
	defer iw2.Flush()

	tr1 := io.TeeReader(r1, iw1)
	tr2 := io.TeeReader(r2, iw2)

	same, err := DiffReaders(tr1, tr2)
	if err != nil {
		log.Fatal(err)
	}
	return same
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
