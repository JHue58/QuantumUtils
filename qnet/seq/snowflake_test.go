package seq

import (
	"testing"
)

func BenchmarkSnowFlake(b *testing.B) {
	sf := NewSnowFlakeSeqGenerator()
	for i := 0; i < b.N; i++ {
		sf.NextSeq()
	}
}
