package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/mjibson/go-dsp/fft"
	"io/ioutil"
	"math"
	"math/cmplx"
)

// pcm_s16le PCM means "traditional wave like format" (raw bytes, basically). 16 means 16 bits per sample, "le" means "little endian", s means "signed", u would mean "unsigned"

const (
	fftSize               = 1024.0
	smoothingTimeConstant = 0.2
	maxDecibels           = -20.0 // -20.0 works better
	minDecibels           = -80.0 // -80.0 works better
	ucharMax              = 255.0
	threshold             = 14
	sampleRate            = 44100.0
	minimumTime           = 80
)

var (
	nonMaxMin int
	maxMin    int
)

func main() {
	rawFile, err := ioutil.ReadFile("./audio/02_How_Did_I_Get_Here.wav")
	if err != nil {
		panic(err)
	}
	if len(rawFile) < 44 {
		panic("INVALID_WAV_HEADER")
	}
	fmt.Println(string(rawFile[0:4]))     // Marks the file as a riff file. Characters are each 1 byte long.
	fmt.Println(getInt32(rawFile[4:8]))   // Size of the overall file - 8 bytes, in bytes (32-bit integer). Typically, you'd fill this in after creation.
	fmt.Println(string(rawFile[8:12]))    // File Type Header. For our purposes, it always equals "WAVE".
	fmt.Println(string(rawFile[12:16]))   // Format chunk marker. Includes trailing null
	fmt.Println(getInt32(rawFile[16:20])) // Length of format data as listed above
	fmt.Println(getInt16(rawFile[20:22])) // Type of format (1 is PCM) - 2 byte integer
	fmt.Println(getInt16(rawFile[22:24])) // Number of Channels - 2 byte integer
	fmt.Println(getInt32(rawFile[24:28])) // Sample Rate - 32 byte integer. Common values are 44100 (CD), 48000 (DAT). Sample Rate = Number of Samples per second, or Hertz.
	fmt.Println(getInt32(rawFile[28:32])) // (Sample Rate * BitsPerSample * Channels) / 8.
	fmt.Println(getInt16(rawFile[32:34])) // (BitsPerSample * Channels) / 8.1 - 8 bit mono2 - 8 bit stereo/16 bit mono4 - 16 bit stereo
	fmt.Println(getInt16(rawFile[34:36])) // Bits per sample
	fmt.Println(string(rawFile[36:40]))   // "list" chunk header.
	listSize := getInt32(rawFile[40:44])
	fmt.Println(listSize)                                       // Size of the list section
	fmt.Println(string(rawFile[44:48]))                         // list type ID
	fmt.Println(string(rawFile[48 : 48+listSize-4]))            // list data
	fmt.Println(string(rawFile[48+listSize-4 : 48+listSize]))   // begin of "data" section
	fmt.Println(getInt32(rawFile[48+listSize : 48+listSize+4])) // data section size

	var audio []int16
	for i := 48 + int(listSize) + 4; i <= len(rawFile); i += 4 {
		audio = append(audio, (getInt16(rawFile[i-4:i-2])+getInt16(rawFile[i-2:i]))/2)
	}
	fft := doFFT(audio)
	fmt.Println(len(audio), len(fft))
	fmt.Println(nonMaxMin, maxMin)

	var avg []byte
	for i := range fft {
		avg = append(avg, getAvg(fft[i]))
	}

	var prev byte = avg[0]
	var currT, prevT int
	var result []int
	for i := 0; i < len(avg); i += 1 {
		currT = int(float64(i) * fftSize * 1000 / sampleRate)
		if currT-prevT > minimumTime && math.Abs(float64(avg[i])-float64(prev)) > threshold {
			result = append(result, currT-prevT)
			prevT = currT
		}
		prev = avg[i]
	}
	result = append(result, currT-prevT)
	fmt.Println(result)
}

func getAvg(data []byte) byte {
	var total int
	for _, b := range data {
		total += int(b)
	}
	return byte(total / len(data))
}

func doFFT(data []int16) (result [][]byte) {
	var dataF []float64
	prev := make([]float64, fftSize/2) // stub out all zero values
	for _, f := range data {
		if len(dataF) == fftSize {
			var abs []float64
			comp := fft.FFTReal(applyWindow(dataF))
			comp[0] = complex(real(comp[0]), 0)
			for j := 0; j < fftSize/2; j++ {
				abs = append(abs, smoothingTimeConstant*prev[j]+(1-smoothingTimeConstant)*cmplx.Abs(comp[j])/fftSize)
			}
			result = append(result, convertToUnsignedBytes(abs))
			prev = abs
			dataF = nil
		}
		dataF = append(dataF, float64(f))
	}
	return
}

func convertToUnsignedBytes(data []float64) (result []byte) {
	rangeScaleFactor := 1 / (maxDecibels - minDecibels)
	for _, linearValue := range data {
		var dbMag float64
		if linearValue == 0 {
			dbMag = minDecibels
		} else {
			dbMag = linearToDecibels(linearValue)
		}

		scaledValue := ucharMax * (dbMag - minDecibels) * rangeScaleFactor
		if scaledValue < 0 {
			scaledValue = 0
		}
		if scaledValue > ucharMax {
			scaledValue = ucharMax
		}
		if scaledValue > 0 && scaledValue < ucharMax {
			nonMaxMin++
		} else {
			maxMin++
		}
		result = append(result, byte(scaledValue))
	}
	return
}

func getInt16(data []byte) (ret int16) {
	if len(data) != 2 {
		panic(fmt.Errorf("incorrect number of bytes for int16: %d", len(data)))
	}
	buf := bytes.NewBuffer(data)
	binary.Read(buf, binary.LittleEndian, &ret)
	return
}

func getInt32(data []byte) (ret int32) {
	if len(data) != 4 {
		panic(fmt.Errorf("incorrect number of bytes for int32: %d", len(data)))
	}
	buf := bytes.NewBuffer(data)
	binary.Read(buf, binary.LittleEndian, &ret)
	return
}

func linearToDecibels(linear float64) float64 {
	if linear == 0 {
		return -1000
	}
	return 20 * math.Log10(linear)
}

func applyWindow(data []float64) []float64 {
	// Blackman window
	alpha := 0.16
	a0 := 0.5 * (1 - alpha)
	a1 := 0.5
	a2 := 0.5 * alpha

	for i := 0; i < len(data); i++ {
		x := float64(i) / float64(len(data))
		window := a0 - a1*math.Cos(2*math.Pi*x) + a2*math.Cos(2*math.Pi*x)
		data[i] *= window
	}
	return data
}