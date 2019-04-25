package cachetable

import (
	"errors"
	"math/bits"
)

const MaxUint uint = 1<<bits.UintSize - 1

// Node which is stored at each level
type Node struct {
	key         string
	Value       interface{}
	create_time uint
}

// CacheTable implemented with a fixed bucketcapacity.
// removes oldest element in bucket once bucket reaches bucketcapacity
type CacheTable struct {
	current_time   uint
	bucketcapacity int
	numbuckets     int
	count          int
	buckets        [][]Node
}

/** PRIVATE METHODS **/

// returns the index at which the key needs to go
func (h *CacheTable) getIndex(key string) int {
	return int(hash(key)) % h.numbuckets
}

// Implements the Jenkins hash function
func hash(key string) uint32 {
	var h uint32
	for _, c := range key {
		h += uint32(c)
		h += (h << 10)
		h ^= (h >> 6)
	}
	h += (h << 3)
	h ^= (h >> 11)
	h += (h << 15)
	return h
}

/** PUBLIC METHODS **/

// Len returns the count of the elements in the cachetable
func (h *CacheTable) Len() int {
	return h.count
}

// BucketCapacity returns the bucket size of the cachetable
func (h *CacheTable) BucketCapacity() int {
	return h.bucketcapacity
}

// Capacity returns the overall size of the cachetable
func (h *CacheTable) Capacity() int {
	return h.bucketcapacity * h.numbuckets
}

// NewCacheTable is the constuctor that returns a new CacheTable of a fixed size
// returns an error when a size of 0 is provided
func NewCacheTable(numbuckets, bucketcapacity int, preallocatemem bool) (*CacheTable, error) {
	if bucketcapacity < 1 {
		return nil, errors.New("bucketcapacity of cachetable has to be => 1")
	}
	if numbuckets < 1 {
		return nil, errors.New("number of buckets of cachetable has to be => 1")
	}
	h := new(CacheTable)
	h.bucketcapacity = bucketcapacity
	h.numbuckets = numbuckets
	h.count = 0
	h.current_time = 0
	h.buckets = make([][]Node, numbuckets)
	initial_bucket_capacity := 0
	if preallocatemem {
		initial_bucket_capacity = bucketcapacity
	}
	for i := range h.buckets {
		h.buckets[i] = make([]Node, 0, initial_bucket_capacity)
	}
	return h, nil
}

// Get returns the value associated with a key in the cachetable,
// and an error indicating whether the value exists
func (h *CacheTable) Get(key string) (*Node, bool) {
	index := h.getIndex(key)
	chain := h.buckets[index]
	for _, node := range chain {
		if node.key == key {
			return &node, true
		}
	}
	return nil, false
}

func findElementOrOldestIndex(chain []Node, begin_of_time uint, key string) (oldest_node_seen_index int, elem_found bool) {
	oldest_node_seen_time := MaxUint
	oldest_node_seen_index = 0
	elem_found = false
	for i, node := range chain {
		// if found, update the value
		if node.key == key {
			return i, true
		}
		//since we have not assigned current_time to a new node yet,
		//h.current_time is currently also the beginning-of-time after
		//a possible roll-over of uint
		node_corrected_create_time := node.create_time - begin_of_time
		// '<='' because at first we just assume 0 is the oldest and we might actually encounter a node with create_time=MaxUInt
		// and since each create_time should be unique, we set oldest_node_seen_index to that node then
		if node_corrected_create_time <= oldest_node_seen_time {
			oldest_node_seen_index = i
			oldest_node_seen_time = node_corrected_create_time
		}
	}
	return
}

// Set the value for an associated key in the cachetable
// this always success as it will just overwrite the oldest element in the bucket
func (h *CacheTable) Set(key string, value interface{}) {
	index := h.getIndex(key)
	chain := h.buckets[index]

	index_for_new_data, elem_found := findElementOrOldestIndex(chain, h.current_time, key)

	// if key doesn't exist, add it to the cachetable
	newnode := Node{key: key, Value: value, create_time: h.current_time}
	h.current_time++ //increment cachetable insert time

	// if key was found, overwrite it
	// if bucket is full overwrite oldest element
	if elem_found || len(chain) >= h.bucketcapacity {
		chain[index_for_new_data] = newnode
	} else {
		// there's enough space, let's append the node
		chain = append(chain, newnode)
		h.buckets[index] = chain
		h.count++
	}
	return
}

// Delete the value associated with key in the cachetable
func (h *CacheTable) Delete(key string) (*Node, bool) {
	index := h.getIndex(key)
	chain := h.buckets[index]

	found := false
	var location int
	var mapNode *Node

	// start a search for the key
	for loc, node := range chain {
		if node.key == key {
			found = true
			location = loc
			mapNode = &node
		}
	}

	// if found delete the elem from the slice
	if found {
		h.count--
		N := len(chain) - 1
		chain[location], chain[N] = chain[N], chain[location]
		chain = chain[:N]
		h.buckets[index] = chain
		return mapNode, true
	}

	// if not found return false
	return nil, false
}

// Load returns the load factor of the cachetable
func (h *CacheTable) Load() float32 {
	return float32(h.count) / float32(h.Capacity())
}
