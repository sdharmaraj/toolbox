package data

import (
	"sync"
	"github.com/viant/toolbox"
	"sync/atomic"
)

type field struct {
	Name  string
	Index int
}

type nilGroup int

//CompactedSlice represented a compacted slice to represent object collection
type CompactedSlice struct {
	optimizedStorage bool
	lock             *sync.RWMutex
	fieldNames       map[string]*field
	fields           []*field
	data             [][]interface{}
	size             uint64
}

func (s *CompactedSlice) Size() int {
	return int(atomic.LoadUint64(&s.size))
}

func (s *CompactedSlice) index(fieldName string) int {
	s.lock.RLock()
	f, ok := s.fieldNames[fieldName]
	s.lock.RUnlock()
	if ok {
		return f.Index
	}
	f = &field{Name: fieldName, Index: len(s.fieldNames)}
	s.lock.Lock()
	defer s.lock.Unlock()
	s.fieldNames[fieldName] = f
	s.fields = append(s.fields, f)
	atomic.StoreUint64(&s.size, 0)
	return f.Index
}

func expandIfNeeded(size int, data []interface{}) []interface{} {
	if size >= len(data) {
		for i := len(data); i < size; i++ {
			data = append(data, nil)
		}
	}
	return data
}

func (s *CompactedSlice) compressStorage(data []interface{}) []interface{} {
	var compressed = make([]interface{}, 0)
	var nilCount = 0
	for _, item := range data {
		if item != nil {
			switch nilCount {
			case 0:
			case 1:
				compressed = append(compressed, nil)
			default:
				compressed = append(compressed, nilGroup(nilCount))
			}
			compressed = append(compressed, item)
			nilCount = 0
			continue
		}
		nilCount++
	}
	return compressed
}

func (s *CompactedSlice) uncompressStorage(in, out []interface{}) {
	var index = 0
	for i := 0; i < len(in); i++ {
		var item = in[i]
		nilGroup, ok := item.(nilGroup)
		if ! ok {
			out[index] = item
			index++
			continue
		}
		for j := 0; j < int(nilGroup); j++ {
			out[index] = nil
			index++
		}
	}
	for i := index; i<len(out);i++ {
		out[i] = nil
	}
}

func (s *CompactedSlice) Add(data map[string]interface{}) {
	var initSize = len(s.fieldNames)
	if initSize < len(data) {
		initSize = len(data)
	}
	var record = make([]interface{}, initSize)
	for k, v := range data {
		i := s.index(k)
		if ! (i < len(record)) {
			record = expandIfNeeded(i+1, record)
		}
		record[i] = v
		atomic.AddUint64(&s.size, 1)
	}
	if s.optimizedStorage {
		record = s.compressStorage(record)
	}
	s.data = append(s.data, record)
}

func (s *CompactedSlice) Range(handler func(item interface{}) (bool, error)) error {
	s.lock.Lock()
	fields := s.fields
	data := s.data
	s.data = [][]interface{}{}
	s.lock.Unlock()

	var record = make([]interface{}, len(s.fields))
	for _, item := range data {

		if s.optimizedStorage {
			s.uncompressStorage(item, record)
		} else {
			record = item
		}

		var aMap = map[string]interface{}{}
		for _, field := range fields {
			index := field.Index
			var value = record[index]
			if value == nil {
				continue
			}
			if toolbox.IsString(value) {
				if toolbox.AsString(value) == "" {
					continue
				}
			} else if toolbox.IsInt(value) {
				if toolbox.AsInt(value) == 0 {
					continue
				}
			} else if toolbox.IsFloat(value) {
				if toolbox.AsFloat(value) == 0.0 {
					continue
				}
			}
			aMap[field.Name] = value
		}
		if next, err := handler(aMap); ! next || err != nil {
			return err
		}
	}
	return nil
}

func NewCompactedSlice(optimizedStorage bool) *CompactedSlice {
	return &CompactedSlice{
		optimizedStorage: optimizedStorage,
		fields:           make([]*field, 0),
		fieldNames:       make(map[string]*field),
		data:             make([][]interface{}, 0),
		lock:             &sync.RWMutex{},
	}
}
