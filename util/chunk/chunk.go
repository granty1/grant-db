package chunk

// Chunk define the struct
// store data in Apache Arrow Format
type Chunk struct {
	// Indicate which rows are selected
	sel []int
	//TODO Grant: columns

	numVirtualRows int

	capacity int

	requiredRows int
}
