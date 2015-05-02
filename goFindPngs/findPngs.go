package main;

import (
	"fmt"
	"flag"
	"bufio"
	"os"
	"image/png"
	"strconv"
)

var fileName = flag.String("f", "", "filename")

func main(){
	flag.Parse();

	file, err := os.Open(*fileName)
	if err!=nil {
		panic(err)
	}
	defer file.Close()

	r := bufio.NewReader(file)

	var bcount int64 = 0
	var detectBytes = []byte{0x89,0x50,0x4e,0x47,0x0d,0x0a}

	for true {
		bval, err := r.ReadByte();
		
		if err != nil {
			fmt.Println(err.Error())
			break;
		}

		bcount+=1;

		if bval == detectBytes[0] {
			r.UnreadByte()
			nextBytes,_ := r.Peek(len(detectBytes))

			if testEq(nextBytes, detectBytes) {
				fmt.Println("Found png, start at:", bcount)
				img, err := png.Decode(r)
				if err == nil {
					fw, err := os.Create("test" + strconv.FormatInt(bcount,10) + ".png" )
					if err == nil {
						defer fw.Close()
						png.Encode(fw,img)
					}
				}
			} else {
				// read byte again to proceed
				r.ReadByte();
			}
		}
	}
	fmt.Printf("done, read %d bytes",bcount)
}

func testEq(a, b []byte) bool {
    if len(a) != len(b) {
        return false
    }

    for i := range a {
        if a[i] != b[i] {
            return false
        }
    }

    return true
}