// +build go1.12

package gmf

/*

#cgo pkg-config: libswscale

#include "libavutil/frame.h"
#include "libswscale/swscale.h"

void gmf_rotate_rgba_180(uint8_t** rgba, int width, int height) {
	uint32_t* src = (uint32_t*)rgba[0];
	const int stride = width;
	int idx1;
	int idx2;
	for (int i = 0, j = height - 1; i < j; i++, j--) {
		for (int x = 0, y = width - 1; x < width; x++, y--) {
			idx1 = (i * stride) + x;
			idx2 = (j * stride) + y;
			uint32_t tmp = src[idx1];
			src[idx1] = src[idx2];
			src[idx2] = tmp;
		}
	}
}
void gmf_scale_rgba(struct SwsContext* ctx, uint8_t* rgba, int width, int height, struct AVFrame* frame) {
	if (!frame) {
		fprintf(stderr, "[gmf_scale_rgba] frame is NULL\n");
	}
	const uint8_t *const inData[1] = { rgba };
	const int inStride[1] = { 4 * width };
	sws_scale(ctx, inData, inStride, 0, height, frame->data, frame->linesize);
}
*/
import "C"

import (
	"fmt"
	"image"
	"unsafe"
)

var (
	SWS_FAST_BILINEAR int = C.SWS_FAST_BILINEAR
	SWS_BILINEAR      int = C.SWS_BILINEAR
	SWS_BICUBIC       int = C.SWS_BICUBIC
	SWS_X             int = C.SWS_X
	SWS_POINT         int = C.SWS_POINT
	SWS_AREA          int = C.SWS_AREA
	SWS_BICUBLIN      int = C.SWS_BICUBLIN
	SWS_GAUSS         int = C.SWS_GAUSS
	SWS_SINC          int = C.SWS_SINC
	SWS_LANCZOS       int = C.SWS_LANCZOS
	SWS_SPLINE        int = C.SWS_SPLINE
)

type SwsCtx struct {
	swsCtx *C.struct_SwsContext
	width  int
	height int
	pixfmt int32
}

func NewSwsCtx(srcW, srcH int, srcPixFmt int32, dstW, dstH int, dstPixFmt int32, method int) (*SwsCtx, error) {
	ctx := C.sws_getContext(
		C.int(srcW),
		C.int(srcH),
		srcPixFmt,
		C.int(dstW),
		C.int(dstH),
		dstPixFmt,
		C.int(method), nil, nil, nil,
	)

	if ctx == nil {
		return nil, fmt.Errorf("error creating sws context\n")
	}

	return &SwsCtx{
		swsCtx: ctx,
		width:  dstW,
		height: dstH,
		pixfmt: dstPixFmt,
	}, nil
}

func NewPicSwsCtx(srcWidth int, srcHeight int, srcPixFmt int32, dst *CodecCtx, method int) *SwsCtx {
	ctx := C.sws_getContext(C.int(srcWidth), C.int(srcHeight), srcPixFmt, C.int(dst.Width()), C.int(dst.Height()), dst.PixFmt(), C.int(method), nil, nil, nil)

	if ctx == nil {
		return nil
	}

	return &SwsCtx{swsCtx: ctx}
}

func (ctx *SwsCtx) Scale(src *Frame, dst *Frame, rotateRgba180 bool) {
	C.sws_scale(
		ctx.swsCtx,
		(**C.uint8_t)(unsafe.Pointer(&src.avFrame.data)),
		(*C.int)(unsafe.Pointer(&src.avFrame.linesize)),
		0,
		C.int(src.Height()),
		(**C.uint8_t)(unsafe.Pointer(&dst.avFrame.data)),
		(*C.int)(unsafe.Pointer(&dst.avFrame.linesize)))
	if rotateRgba180 {
		C.gmf_rotate_rgba_180(
			(**C.uint8_t)(unsafe.Pointer(&dst.avFrame.data)),
			C.int(dst.Width()),
			C.int(dst.Height()))
	}
}

func (this *SwsCtx) ScaleRGBA(src *image.RGBA, dst *Frame) {
	bounds := src.Bounds()
	width := bounds.Max.X
	height := bounds.Max.Y
	C.gmf_scale_rgba(
		this.swsCtx,
		(*C.uint8_t)(unsafe.Pointer(&src.Pix[0])),
		C.int(width),
		C.int(height),
		dst.avFrame)
}

func (ctx *SwsCtx) Free() {
	if ctx.swsCtx != nil {
		C.sws_freeContext(ctx.swsCtx)
	}
}

func DefaultRescaler(ctx *SwsCtx, frames []*Frame, rotateRgba180 bool) ([]*Frame, error) {
	var (
		result []*Frame = make([]*Frame, 0)
		tmp    *Frame
		err    error
	)

	for i, _ := range frames {
		tmp = NewFrame().SetWidth(ctx.width).SetHeight(ctx.height).SetFormat(ctx.pixfmt)
		if err = tmp.ImgAlloc(); err != nil {
			return nil, fmt.Errorf("error allocation tmp frame - %s", err)
		}

		ctx.Scale(frames[i], tmp, rotateRgba180)

		tmp.SetPts(frames[i].Pts())
		tmp.SetPktDts(frames[i].PktDts())

		result = append(result, tmp)
	}

	for i := 0; i < len(frames); i++ {
		if frames[i] != nil {
			frames[i].Free()
		}
	}

	return result, nil
}
