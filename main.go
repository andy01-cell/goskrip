package main

import (
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"strings"
)

const threshold = 180

func main() {
	http.HandleFunc("/upload", uploadHandler)
	http.HandleFunc("/bg", bgHandler)
	http.ListenAndServe(":8080", nil)
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	file, _, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "Error retrieving image", http.StatusBadRequest)
		return
	}
	defer file.Close()

	tempFile, err := os.CreateTemp("gambar", "uploaded-*.jpg")
	if err != nil {
		http.Error(w, "Error saving file", http.StatusInternalServerError)
		return
	}
	defer tempFile.Close()

	_, err = io.Copy(tempFile, file)
	if err != nil {
		http.Error(w, "Error copying file", http.StatusInternalServerError)
		return
	}

	inputImage, err := loadImage(tempFile.Name())
	if err != nil {
		http.Error(w, "Error loading image", http.StatusInternalServerError)
		return
	}

	outputImage := removeNoise(inputImage)

	outputFilePath := "output.jpg" // Bisa juga output.png jika ingin PNG
	err = saveImage(outputImage, outputFilePath)
	if err != nil {
		http.Error(w, "Error saving image", http.StatusInternalServerError)
		return
	}

	outputFile, err := os.Open(outputFilePath)
	if err != nil {
		http.Error(w, "Error opening output image", http.StatusInternalServerError)
		return
	}
	defer outputFile.Close()

	w.Header().Set("Content-Type", "image/jpeg") // Bisa juga image/png jika ingin PNG
	_, err = io.Copy(w, outputFile)
	if err != nil {
		http.Error(w, "Error sending response", http.StatusInternalServerError)
		return
	}

	fmt.Println("Noise removal completed and sent to client.")
}

func bgHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	file, _, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "Error retrieving image", http.StatusBadRequest)
		return
	}
	defer file.Close()

	tempFile, err := os.CreateTemp("gambar", "uploaded-*.png")
	if err != nil {
		http.Error(w, "Error saving file", http.StatusInternalServerError)
		return
	}
	defer tempFile.Close()

	_, err = io.Copy(tempFile, file)
	if err != nil {
		http.Error(w, "Error copying file", http.StatusInternalServerError)
		return
	}

	inputImage, err := loadImage(tempFile.Name())
	if err != nil {
		http.Error(w, "Error loading image", http.StatusInternalServerError)
		return
	}

	outputImage := removeBackground(inputImage)

	outputFilePath := "output.png" // PNG format to support transparency
	err = saveImage(outputImage, outputFilePath)
	if err != nil {
		http.Error(w, "Error saving image", http.StatusInternalServerError)
		return
	}

	outputFile, err := os.Open(outputFilePath)
	if err != nil {
		http.Error(w, "Error opening output image", http.StatusInternalServerError)
		return
	}
	defer outputFile.Close()

	w.Header().Set("Content-Type", "image/png") // PNG to support transparency
	_, err = io.Copy(w, outputFile)
	if err != nil {
		http.Error(w, "Error sending response", http.StatusInternalServerError)
		return
	}

	fmt.Println("Background removal completed and sent to client.")
}

func loadImage(filePath string) (image.Image, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, format, err := image.Decode(file)
	if err != nil {
		return nil, err
	}

	fmt.Println("Loaded image with format:", format)
	return img, nil
}

func saveImage(img image.Image, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	switch ext := getExtension(filePath); ext {
	case "jpg", "jpeg":
		err = jpeg.Encode(file, img, nil)
	case "png":
		err = png.Encode(file, img)
	default:
		return fmt.Errorf("unsupported file format: %s", ext)
	}

	if err != nil {
		return err
	}

	return nil
}

func getExtension(filePath string) string {
	parts := strings.Split(filePath, ".")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return ""
}

func removeNoise(inputImage image.Image) image.Image {
	bounds := inputImage.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	outputImage := image.NewRGBA(bounds)

	kernelSize := 3
	halfKernelSize := kernelSize / 2

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			var totalR, totalG, totalB uint32

			for ky := -halfKernelSize; ky <= halfKernelSize; ky++ {
				for kx := -halfKernelSize; kx <= halfKernelSize; kx++ {
					nx := x + kx
					ny := y + ky

					if nx >= 0 && nx < width && ny >= 0 && ny < height {
						r, g, b, _ := inputImage.At(nx, ny).RGBA()

						totalR += r
						totalG += g
						totalB += b
					}
				}
			}

			avgR := totalR / uint32(kernelSize*kernelSize)
			avgG := totalG / uint32(kernelSize*kernelSize)
			avgB := totalB / uint32(kernelSize*kernelSize)

			outputImage.Set(x, y, color.RGBA{
				uint8(avgR >> 8),
				uint8(avgG >> 8),
				uint8(avgB >> 8),
				255, // Alpha channel
			})
		}
	}

	return outputImage
}

func removeBackground(img image.Image) image.Image {
	bounds := img.Bounds()
	newImg := image.NewRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()

			r8 := uint8(r >> 8)
			g8 := uint8(g >> 8)
			b8 := uint8(b >> 8)
			a8 := uint8(a >> 8)	

			if r8 > threshold && g8 > threshold && b8 > threshold && a8 > threshold {
				newImg.Set(x, y, color.RGBA{0, 0, 0, 0})
			} else {
				newImg.Set(x, y, color.RGBA{r8, g8, b8, a8})
			}
		}
	}

	return newImg
}
