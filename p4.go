/*
Package p4 wraps the Perforce Helix Core command line.

It assumes p4 or p4.exe is in the PATH.
It uses the p4 -G global option which returns Python marshalled dictionary objects.

p4 Python parsing module is based on: https://github.com/hambster/gopymarshal
*/
package p4

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os/exec"
	"regexp"
	"strings"

	"encoding/json"
	"errors"
)

// P4 - environment for P4
type P4 struct {
	port   string
	user   string
	client string
}

// NewP4 - create and initialise properly
func NewP4() *P4 {
	var p4 P4
	return &p4
}

// NewP4Params - create and initialise with params
func NewP4Params(port string, user string, client string) *P4 {
	var p4 P4
	p4.port = port
	p4.user = user
	p4.client = client
	return &p4
}

// RunBytes - runs p4 command and returns []byte output
func (p4 *P4) RunBytes(args []string) ([]byte, error) {
	cmd := exec.Command("p4", args...)

	data, err := cmd.CombinedOutput()
	if err != nil {
		return data, err
	}
	return data, nil
}

// Get options that go before the p4 command
func (p4 *P4) getJOptions() []string {
	opts := []string{"-Mj", "-ztag"}

	if p4.port != "" {
		opts = append(opts, "-p", p4.port)
	}
	if p4.user != "" {
		opts = append(opts, "-u", p4.user)
	}
	if p4.client != "" {
		opts = append(opts, "-c", p4.client)
	}
	return opts
}

// Get options that go before the p4 command
func (p4 *P4) getOptionsNonMarshal() []string {
	opts := []string{}

	if p4.port != "" {
		opts = append(opts, "-p", p4.port)
	}
	if p4.user != "" {
		opts = append(opts, "-u", p4.user)
	}
	if p4.client != "" {
		opts = append(opts, "-c", p4.client)
	}
	return opts
}

// Runner is an interface to make testing p4 commands more easily
type Runner interface {
	Run([]string) ([]map[string]string, error)
}

// Run - runs p4 command and returns map
func (p4 *P4) Run(args []string) ([]map[string]string, error) {
	opts := p4.getJOptions()
	args = append(opts, args...)
	cmd := exec.Command("p4", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	mainerr := cmd.Run()
	// May not be the correct place to do this
	// But we are ignoring the actual error otherwise
	if stderr.Len() > 0 {
		return nil, errors.New(stderr.String())
	}
  
	results := make([]map[string]string, 0)
	jdecoder := json.NewDecoder(&stdout)
	for {
		line, _, _ := buf.ReadLine()
		r := make(map[string]string)
		err := jdecoder.Decode(&r)
		if err == io.EOF {
			break
		}
		if err == nil {
			if r == nil {
				// End of object
				break
			}
		} else {
			if mainerr == nil {
				mainerr = err
			}
			// The stdout contains context regarding the error
			// in some cases, e.g. password expiring so we should
			// also populate the results to convey the error details
			// to the caller. Otherwise they'll just see an "exit
			// status 1" in the error and empty results.
			if r != nil {
				results = append(results, r)
			}
			break
		}
		results = append(results, r)
	}
	return results, mainerr
}

// parseError turns perforce error messages into go error's
func parseError(res map[string]string) error {
	var err error
	var e string
	if v, ok := res["data"]; ok {
		e = v
	} else {
		// I don't know if we can get in this situation
		e = fmt.Sprintf("Failed to parse error %v", err)
		return errors.New(e)
	}
	// Search for non-existent depot error
	nodepot, err := regexp.Match(`must refer to client`, []byte(e))
	if err != nil {
		return err // Do we need to return (error, error) for real error and parsed one?
	}
	if nodepot {
		path := strings.Split(e, " - must")[0]
		return errors.New("P4Error -> No such area '" + path + "', please check your path")
	}
	err = fmt.Errorf("P4Error -> %s", e)
	return err
}

// Assume multiline entries should be on seperate lines
func formatSpec(specContents map[string]string) string {
	var output bytes.Buffer
	for k, v := range specContents {
		if strings.Contains(v, "\n") {
			output.WriteString(fmt.Sprintf("%s:", k))
			lines := strings.Split(v, "\n")
			for i := range lines {
				if len(strings.TrimSpace(lines[i])) > 0 {
					output.WriteString(fmt.Sprintf("\n %s", lines[i]))
				}
			}
			output.WriteString("\n\n")
		} else {
			output.WriteString(fmt.Sprintf("%s: %s\n\n", k, v))
		}
	}
	return output.String()
}

// Save - runs p4 -i for specified spec returns result
func (p4 *P4) Save(specName string, specContents map[string]string, args []string) ([]map[string]string, error) {
	opts := p4.getJOptions()
	nargs := []string{specName, "-i"}
	nargs = append(nargs, args...)
	args = append(opts, nargs...)

	log.Println(args)
	cmd := exec.Command("p4", args...)
	var stdout, stderr bytes.Buffer
	stdin, err := cmd.StdinPipe()
	if err != nil {
		fmt.Println("An error occured: ", err)
	}
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	mainerr := cmd.Start()
	if mainerr != nil {
		fmt.Println("An error occured: ", mainerr)
	}
	spec := formatSpec(specContents)
	log.Println(spec)
	io.WriteString(stdin, spec)
	stdin.Close()
	cmd.Wait()

	results := make([]map[string]string, 0)
	for {
		r := make(map[string]string)
		err := json.NewDecoder(&stdout).Decode(&r)
		if err == io.EOF {
			break
		}
		if err == nil {
			if r == nil {
				// End of object
				break
			}
			results = append(results, r)
		} else {
			if mainerr == nil {
				mainerr = err
			}
			break
		}
	}
	return results, mainerr
}

// The Save() func doesn't work as it needs the data marshalled instead of
// map[string]string
// This is a quick fix, the real fix is writing a marshal() function or try
// using gopymarshal
func (p4 *P4) SaveTxt(specName string, specContents map[string]string, args []string) (string, error) {
	opts := p4.getOptionsNonMarshal()
	nargs := []string{specName, "-i"}
	nargs = append(nargs, args...)
	args = append(opts, nargs...)

	log.Println(args)
	cmd := exec.Command("p4", args...)
	var stdout, stderr bytes.Buffer
	stdin, err := cmd.StdinPipe()
	if err != nil {
		fmt.Println("An error occured: ", err)
	}
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	mainerr := cmd.Start()
	if mainerr != nil {
		fmt.Println("An error occured: ", mainerr)
	}
	spec := formatSpec(specContents)
	log.Println(spec)
	io.WriteString(stdin, spec)
	// Need to explicitly call this for the command to fire
	stdin.Close()
	cmd.Wait()

	e, err := ioutil.ReadAll(&stderr)
	if err != nil {
		fmt.Println("An error occured: ", err)
	}
	log.Println(e)
	if len(e) > 0 {
		return "", errors.New(string(e))
	}
	x, err := ioutil.ReadAll(&stdout)
	if err != nil {
		fmt.Println("An error occured: ", err)
	}
	s := string(x)
	log.Println(s)
	return s, mainerr
}
