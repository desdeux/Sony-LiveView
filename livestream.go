package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	lv "github.com/desdeux/sony-liveview/liveview"
	"github.com/veandco/go-sdl2/img"
	"github.com/veandco/go-sdl2/sdl"
)

var (
	liveviewURL = "http://192.168.122.1:8080"
)

func main() {
	cmdArgs := os.Args
	if len(cmdArgs) == 1 {
		fmt.Println("You should provide a liveview URL in order to start, for example: livestream http://192.168.122.1:8080")
		return
	}

	liveviewURL = cmdArgs[1]
	//Placeholder size
	imageWidth := int32(640)
	imageHeight := int32(480)

	liveview, err := lv.Start(liveviewURL)
	if err != nil {
		log.Printf("%s\n", err)
		os.Exit(1)
	}

	defer liveview.Stop()

	err = sdl.Init(sdl.INIT_EVERYTHING)
	if err != nil {
		log.Printf("Failed to initialize sdl: %s\n", err)
		os.Exit(1)
	}

	window, err := sdl.CreateWindow("Liveview stream", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, 640, 424, sdl.WINDOW_SHOWN)
	if err != nil {
		log.Printf("Failed to create renderer: %s\n", err)
		os.Exit(2)
	}

	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED)
	if err != nil {
		log.Printf("Failed to create renderer: %s\n", err)
		os.Exit(3)
	}

	renderer.Clear()
	img.Init(img.INIT_JPG)
	sdl.SetHint(sdl.HINT_RENDER_SCALE_QUALITY, "1")

	tempBytes, err := ioutil.ReadFile("placeholder.jpg")
	if err != nil {
		log.Printf("Failed to load placeholder: %s\n", err)
		os.Exit(4)
	}
	tempRWops, err := sdl.RWFromMem(tempBytes)
	if err != nil {
		log.Printf("Failed to RWFromMem: %s\n", err)
		os.Exit(4)
	}
	surfaceImg, err := img.LoadJPGRW(tempRWops)
	if err != nil {
		log.Printf("Failed to load image: %s\n", err)
		os.Exit(4)
	}

	textureImg, err := renderer.CreateTextureFromSurface(surfaceImg)
	if err != nil {
		log.Printf("Failed to create texture: %s\n", err)
		os.Exit(5)
	}
	surfaceImg.Free()

	var event sdl.Event

	isRunning := true
	for isRunning {
		err = liveview.Connect()
		if err != nil {
			log.Println(err)
			continue
		}

		for {
			imgData, err := liveview.FetchFrame()
			if err != nil {
				log.Printf("%s\n", err)
				break
			}
			for event = sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
				switch t := event.(type) {
				case *sdl.QuitEvent:
					isRunning = false
				case *sdl.KeyboardEvent:
					if t.Keysym.Sym == sdl.K_ESCAPE {
						isRunning = false
					}
				}
			}

			renderer.SetDrawColor(255, 255, 55, 255)
			renderer.Clear()
			err = renderer.Copy(textureImg, nil, &sdl.Rect{0, 0, imageWidth, imageHeight})
			if err != nil {
				log.Printf("Failed to copy texture: %s\n", err)
			}

			renderer.Present()

			textureImg.Destroy()

			tempRWops, err = sdl.RWFromMem(imgData)
			if err != nil {
				log.Printf("Failed to RWFromMem: %s\n", err)
				os.Exit(4)
			}
			surfaceImg, err = img.LoadJPGRW(tempRWops)
			if err != nil {
				log.Printf("Failed to load image: %s\n", err)
				os.Exit(4)
			}

			imageWidth = surfaceImg.W
			imageHeight = surfaceImg.H

			textureImg, err = renderer.CreateTextureFromSurface(surfaceImg)
			if err != nil {
				log.Printf("Failed to create texture: %s\n", err)
				os.Exit(5)
			}
			surfaceImg.Free()

			time.Sleep(time.Millisecond * 10)

		}
		if isRunning {
			log.Println("Updating stream...")
		}
	}

	textureImg.Destroy()
	img.Quit()
	renderer.Destroy()
	window.Destroy()

	sdl.Quit()
}
