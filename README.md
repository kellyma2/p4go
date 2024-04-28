# p4go
Go interface library to use Perforce Helix Core command line (p4 or p4.exe)

This is a fork of Robert Cowhams great work https://github.com/rcowham/go-libp4 but converted to use p4 -Mj, json output.

The advantage of json is the ease of use of the results, which are of type map[string]string rather than map[interface{}]interface{}
