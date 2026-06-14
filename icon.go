package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
)

// generateIcon draws a simple folder icon and wraps it in a minimal
// ICO container (PNG-compressed, supported since Windows Vista) so it
// can be used as a system tray icon without shipping a binary asset.
func generateIcon() []byte {
	const size = 32
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	tabColor := color.RGBA{0x29, 0xB6, 0xF6, 0xFF}
	folderColor := color.RGBA{0x4F, 0xC3, 0xF7, 0xFF}

	for y := 4; y < 9; y++ {
		for x := 4; x < 14; x++ {
			img.Set(x, y, tabColor)
		}
	}
	for y := 8; y < 26; y++ {
		for x := 3; x < 29; x++ {
			img.Set(x, y, folderColor)
		}
	}

	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, img); err != nil {
		return nil
	}
	pngBytes := pngBuf.Bytes()

	var ico bytes.Buffer
	binary.Write(&ico, binary.LittleEndian, uint16(0)) // reserved
	binary.Write(&ico, binary.LittleEndian, uint16(1)) // type: icon
	binary.Write(&ico, binary.LittleEndian, uint16(1)) // image count

	ico.WriteByte(byte(size)) // width
	ico.WriteByte(byte(size)) // height
	ico.WriteByte(0)          // color palette
	ico.WriteByte(0)          // reserved
	binary.Write(&ico, binary.LittleEndian, uint16(1))             // planes
	binary.Write(&ico, binary.LittleEndian, uint16(32))            // bits per pixel
	binary.Write(&ico, binary.LittleEndian, uint32(len(pngBytes))) // image size
	binary.Write(&ico, binary.LittleEndian, uint32(22))            // image offset

	ico.Write(pngBytes)

	return ico.Bytes()
}
