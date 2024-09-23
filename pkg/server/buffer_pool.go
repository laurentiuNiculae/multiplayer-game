package server

import flatbuffers "github.com/google/flatbuffers/go"

type BuilderPool struct {
	buffers []*flatbuffers.Builder
	isFree  []bool
}

func NewBuilderPool(size, count int) *BuilderPool {
	buffers := make([]*flatbuffers.Builder, count)
	isFree := make([]bool, count)

	for i := range buffers {
		buffers[i] = flatbuffers.NewBuilder(size)
		isFree[i] = true
	}

	return &BuilderPool{buffers: buffers, isFree: isFree}
}

func (bp *BuilderPool) GetFreeBuilder() *flatbuffers.Builder {
	for i := range bp.buffers {
		if bp.isFree[i] {
			bp.isFree[i] = false

			return bp.buffers[i]
		}
	}

	builder := flatbuffers.NewBuilder(512)

	bp.buffers = append(bp.buffers, builder)
	bp.isFree = append(bp.isFree, true)

	return builder
}

func (bp *BuilderPool) Reset() {
	for i := range bp.buffers {
		if !bp.isFree[i] {
			bp.isFree[i] = true
			bp.buffers[i].Reset()
		}
	}
}
