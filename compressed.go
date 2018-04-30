package sparse

import (
	"github.com/james-bowman/sparse/blas"
	"gonum.org/v1/gonum/mat"
)

var (
	csr *CSR

	_ Sparser       = csr
	_ TypeConverter = csr

	_ mat.Mutable = csr

	_ mat.ColViewer = csr
	_ mat.RowViewer = csr

	_ mat.RowNonZeroDoer = csr

	csc *CSC

	_ Sparser       = csc
	_ TypeConverter = csc

	_ mat.Mutable = csc

	_ mat.ColViewer = csc
	_ mat.RowViewer = csc

	_ mat.ColNonZeroDoer = csc
)

// CSR is a Compressed Sparse Row format sparse matrix implementation (sometimes called Compressed Row
// Storage (CRS) format) and implements the Matrix interface from gonum/matrix.  This allows large sparse
// (mostly zero values) matrices to be stored efficiently in memory (only storing non-zero values).
// CSR matrices are poor for constructing sparse matrices incrementally but very good for arithmetic operations.
// CSR, and their sibling CSC, matrices are similar to COOrdinate matrices except the row index slice is
// compressed.  Rather than storing the row indices of each non zero values (length == NNZ) each element, i,
// of the slice contains the cumulative count of non zero values in the matrix up to row i-1 of the matrix.
// In this way, it is possible to address any element, i j, in the matrix with the following:
//
// 		for k := c.indptr[i]; k < c.indptr[i+1]; k++ {
//			if c.ind[k] == j {
//				return c.data[k]
//			}
//		}
//
// It should be clear that CSR is like CSC except the slices are row major order rather than column major and
// CSC is essentially the transpose of a CSR.
// As this type implements the gonum mat.Matrix interface, it may be used with any of the Gonum mat
// functions that accept Matrix types as parameters in place of other matrix types included in the Gonum
// mat package e.g. mat.Dense.
type CSR struct {
	matrix blas.SparseMatrix
}

// NewCSR creates a new Compressed Sparse Row format sparse matrix.
// The matrix is initialised to the size of the specified r * c dimensions (rows * columns)
// with the specified slices containing row pointers and cols indexes of non-zero elements
// and the non-zero data values themselves respectively.  The supplied slices will be used as the
// backing storage to the matrix so changes to values of the slices will be reflected in the created matrix
// and vice versa.
func NewCSR(r int, c int, ia []int, ja []int, data []float64) *CSR {
	if uint(r) < 0 {
		panic(mat.ErrRowAccess)
	}
	if uint(c) < 0 {
		panic(mat.ErrColAccess)
	}

	return &CSR{
		matrix: blas.SparseMatrix{
			I: r, J: c,
			Indptr: ia,
			Ind:    ja,
			Data:   data,
		},
	}
}

// Dims returns the size of the matrix as the number of rows and columns
func (c *CSR) Dims() (int, int) {
	return c.matrix.I, c.matrix.J
}

// At returns the element of the matrix located at row i and column j.  At will panic if specified values
// for i or j fall outside the dimensions of the matrix.
func (c *CSR) At(m, n int) float64 {
	return c.matrix.At(m, n)
}

// Set sets the element of the matrix located at row i and column j to value v.  Set will panic if
// specified values for i or j fall outside the dimensions of the matrix.
func (c *CSR) Set(m, n int, v float64) {
	c.matrix.Set(m, n, v)
}

// T transposes the matrix creating a new CSC matrix sharing the same backing data storage but switching
// column and row sizes and index & index pointer slices i.e. rows become columns and columns become rows.
func (c *CSR) T() mat.Matrix {
	return NewCSC(c.matrix.J, c.matrix.I, c.matrix.Indptr, c.matrix.Ind, c.matrix.Data)
}

// NNZ returns the Number of Non Zero elements in the sparse matrix.
func (c *CSR) NNZ() int {
	return len(c.matrix.Data)
}

// RawMatrix returns a pointer to the underlying blas sparse matrix.
func (c *CSR) RawMatrix() *blas.SparseMatrix {
	return &c.matrix
}

// DoNonZero calls the function fn for each of the non-zero elements of the receiver.
// The function fn takes a row/column index and the element value of the receiver at
// (i, j).  The order of visiting to each non-zero element is row major.
func (c *CSR) DoNonZero(fn func(i, j int, v float64)) {
	for i := 0; i < len(c.matrix.Indptr)-1; i++ {
		c.DoRowNonZero(i, fn)
	}
}

// DoRowNonZero calls the function fn for each of the non-zero elements of row i
// in the receiver.  The function fn takes a row/column index and the element value
// of the receiver at (i, j).
func (c *CSR) DoRowNonZero(i int, fn func(i, j int, v float64)) {
	for j := c.matrix.Indptr[i]; j < c.matrix.Indptr[i+1]; j++ {
		fn(i, c.matrix.Ind[j], c.matrix.Data[j])
	}
}

// Clone copies the specified matrix into the receiver
func (c *CSR) Clone(b mat.Matrix) {
	c.matrix.I, c.matrix.J = b.Dims()

	c.matrix.Indptr = make([]int, c.matrix.I+1)

	k := 0
	for i := 0; i < c.matrix.I; i++ {
		c.matrix.Indptr[i] = k
		for j := 0; j < c.matrix.J; j++ {
			if v := b.At(i, j); v != 0 {
				c.matrix.Ind = append(c.matrix.Ind, j)
				c.matrix.Data = append(c.matrix.Data, v)
				k++
			}
		}

	}
	c.matrix.Indptr[c.matrix.I] = k
}

// ToDense returns a mat.Dense dense format version of the matrix.  The returned mat.Dense
// matrix will not share underlying storage with the receiver nor is the receiver modified by this call.
func (c *CSR) ToDense() *mat.Dense {
	mat := mat.NewDense(c.matrix.I, c.matrix.J, nil)

	for i := 0; i < len(c.matrix.Indptr)-1; i++ {
		for j := c.matrix.Indptr[i]; j < c.matrix.Indptr[i+1]; j++ {
			mat.Set(i, c.matrix.Ind[j], c.matrix.Data[j])
		}
	}

	return mat
}

// ToDOK returns a DOK (Dictionary Of Keys) sparse format version of the matrix.  The returned DOK
// matrix will not share underlying storage with the receiver nor is the receiver modified by this call.
func (c *CSR) ToDOK() *DOK {
	dok := NewDOK(c.matrix.I, c.matrix.J)
	for i := 0; i < len(c.matrix.Indptr)-1; i++ {
		for j := c.matrix.Indptr[i]; j < c.matrix.Indptr[i+1]; j++ {
			dok.Set(i, c.matrix.Ind[j], c.matrix.Data[j])
		}
	}

	return dok
}

// ToCOO returns a COOrdinate sparse format version of the matrix.  The returned COO matrix will
// not share underlying storage with the receiver nor is the receiver modified by this call.
func (c *CSR) ToCOO() *COO {
	rows := make([]int, c.NNZ())
	cols := make([]int, c.NNZ())
	data := make([]float64, c.NNZ())

	for i := 0; i < len(c.matrix.Indptr)-1; i++ {
		for j := c.matrix.Indptr[i]; j < c.matrix.Indptr[i+1]; j++ {
			rows[j] = i
			cols[j] = c.matrix.Ind[j]
			data[j] = c.matrix.Data[j]
		}
	}

	coo := NewCOO(c.matrix.I, c.matrix.J, rows, cols, data)

	return coo
}

// ToCSR returns the receiver
func (c *CSR) ToCSR() *CSR {
	return c
}

// ToCSC returns a Compressed Sparse Column sparse format version of the matrix.  The returned CSC matrix
// will not share underlying storage with the receiver nor is the receiver modified by this call.
// NB, the current implementation uses COO as an intermediate format so converts to COO before converting
// to CSC but attempts to reuse memory in the intermediate formats.
func (c *CSR) ToCSC() *CSC {
	return c.ToCOO().ToCSCReuseMem()
}

// ToType returns an alternative format version fo the matrix in the format specified.
func (c *CSR) ToType(matType MatrixType) mat.Matrix {
	return matType.Convert(c)
}

// RowNNZ returns the Number of Non Zero values in the specified row i.  RowNNZ will panic if i is out of range.
func (c *CSR) RowNNZ(i int) int {
	if uint(i) < 0 || uint(i) >= uint(c.matrix.I) {
		panic(mat.ErrRowAccess)
	}
	return c.matrix.Indptr[i+1] - c.matrix.Indptr[i]
}

// RowView slices the Compressed Sparse Row matrix along its primary axis.
// Returns a VecCOO sparse Vector that shares the same storage with
// the receiver for row i.
func (c *CSR) RowView(i int) mat.Vector {
	if i >= c.matrix.I || i < 0 {
		panic(mat.ErrRowAccess)
	}
	return NewVector(
		c.matrix.J,
		c.matrix.Ind[c.matrix.Indptr[i]:c.matrix.Indptr[i+1]],
		c.matrix.Data[c.matrix.Indptr[i]:c.matrix.Indptr[i+1]],
	)
}

// ColView slices the Compressed Sparse Row matrix along its secondary axis.
// Returns a VecDense dense Vector containing a copy of elements of column j.
func (c *CSR) ColView(j int) mat.Vector {
	if j >= c.matrix.J || j < 0 {
		panic(mat.ErrColAccess)
	}
	slice := make([]float64, c.matrix.I)
	for i := range slice {
		slice[i] = c.At(i, j)
	}
	return mat.NewVecDense(c.matrix.I, slice)
}

// ScatterRow returns a slice representing row i of the matrix in dense format.  Row
// is used as the storage for the operation unless it is nil in which case, new
// storage of the correct length will be allocated.  This method will panic if i
// is out of range or row is not the same length as the number of columns in the matrix i.e.
// the correct size to receive the dense representation of the row.
func (c *CSR) ScatterRow(i int, row []float64) []float64 {
	if i >= c.matrix.I || i < 0 {
		panic(mat.ErrRowAccess)
	}
	if row != nil && len(row) != c.matrix.J {
		panic(mat.ErrRowLength)
	}
	if row == nil {
		row = make([]float64, c.matrix.J)
	}
	blas.Ussc(
		c.matrix.Data[c.matrix.Indptr[i]:c.matrix.Indptr[i+1]],
		row,
		1,
		c.matrix.Ind[c.matrix.Indptr[i]:c.matrix.Indptr[i+1]],
	)
	return row
}

// CSC is a Compressed Sparse Column format sparse matrix implementation (sometimes called Compressed Column
// Storage (CCS) format) and implements the Matrix interface from gonum/matrix.  This allows large sparse
// (mostly zero values) matrices to be stored efficiently in memory (only storing non-zero values).
// CSC matrices are poor for constructing sparse matrices incrementally but very good for arithmetic operations.
// CSC, and their sibling CSR, matrices are similar to COOrdinate matrices except the column index slice is
// compressed.  Rather than storing the column indices of each non zero values (length == NNZ) each element, i,
// of the slice contains the cumulative count of non zero values in the matrix up to column i-1 of the matrix.
// In this way, it is possible to address any element, j i, in the matrix with the following:
//
// 		for k := c.indptr[i]; k < c.indptr[i+1]; k++ {
//			if c.ind[k] == j {
//				return c.data[k]
//			}
//		}
//
// It should be clear that CSC is like CSR except the slices are column major order rather than row major and CSC
// is essentially the transpose of a CSR.
// As this type implements the gonum mat.Matrix interface, it may be used with any of the Gonum mat functions
// that accept Matrix types as parameters in place of other matrix types included in the Gonum mat package
// e.g. mat.Dense.
type CSC struct {
	matrix blas.SparseMatrix
}

// NewCSC creates a new Compressed Sparse Column format sparse matrix.
// The matrix is initialised to the size of the specified r * c dimensions (rows * columns)
// with the specified slices containing column pointers and row indexes of non-zero elements
// and the non-zero data values themselves respectively.  The supplied slices will be used as the
// backing storage to the matrix so changes to values of the slices will be reflected in the created matrix
// and vice versa.
func NewCSC(r int, c int, indptr []int, ind []int, data []float64) *CSC {
	if uint(r) < 0 {
		panic(mat.ErrRowAccess)
	}
	if uint(c) < 0 {
		panic(mat.ErrColAccess)
	}

	return &CSC{
		matrix: blas.SparseMatrix{
			I: c, J: r,
			Indptr: indptr,
			Ind:    ind,
			Data:   data,
		},
	}
}

// Dims returns the size of the matrix as the number of rows and columns
func (c *CSC) Dims() (int, int) {
	return c.matrix.J, c.matrix.I
}

// At returns the element of the matrix located at row i and column j.  At will panic if specified values
// for i or j fall outside the dimensions of the matrix.
func (c *CSC) At(m, n int) float64 {
	return c.matrix.At(n, m)
}

// Set sets the element of the matrix located at row i and column j to value v.  Set will panic if
// specified values for i or j fall outside the dimensions of the matrix.
func (c *CSC) Set(m, n int, v float64) {
	c.matrix.Set(n, m, v)
}

// T transposes the matrix creating a new CSR matrix sharing the same backing data storage but switching
// column and row sizes and index & index pointer slices i.e. rows become columns and columns become rows.
func (c *CSC) T() mat.Matrix {
	return NewCSR(c.matrix.I, c.matrix.J, c.matrix.Indptr, c.matrix.Ind, c.matrix.Data)
}

// DoNonZero calls the function fn for each of the non-zero elements of the receiver.
// The function fn takes a row/column index and the element value of the receiver at
// (i, j).  The order of visiting to each non-zero element is column major.
func (c *CSC) DoNonZero(fn func(i, j int, v float64)) {
	for i := 0; i < len(c.matrix.Indptr)-1; i++ {
		c.DoColNonZero(i, fn)
	}
}

// DoColNonZero calls the function fn for each of the non-zero elements of column j
// in the receiver.  The function fn takes a row/column index and the element value
// of the receiver at (i, j).
func (c *CSC) DoColNonZero(j int, fn func(i, j int, v float64)) {
	for i := c.matrix.Indptr[j]; i < c.matrix.Indptr[j+1]; i++ {
		fn(c.matrix.Ind[i], j, c.matrix.Data[i])
	}
}

// NNZ returns the Number of Non Zero elements in the sparse matrix.
func (c *CSC) NNZ() int {
	return len(c.matrix.Data)
}

// RawMatrix returns a pointer to the underlying blas sparse matrix.
func (c *CSC) RawMatrix() *blas.SparseMatrix {
	return &c.matrix
}

// ToDense returns a mat.Dense dense format version of the matrix.  The returned mat.Dense
// matrix will not share underlying storage with the receiver nor is the receiver modified by this call.
func (c *CSC) ToDense() *mat.Dense {
	mat := mat.NewDense(c.matrix.J, c.matrix.I, nil)

	for i := 0; i < len(c.matrix.Indptr)-1; i++ {
		for j := c.matrix.Indptr[i]; j < c.matrix.Indptr[i+1]; j++ {
			mat.Set(c.matrix.Ind[j], i, c.matrix.Data[j])
		}
	}

	return mat
}

// ToDOK returns a DOK (Dictionary Of Keys) sparse format version of the matrix.  The returned DOK
// matrix will not share underlying storage with the receiver nor is the receiver modified by this call.
func (c *CSC) ToDOK() *DOK {
	dok := NewDOK(c.matrix.J, c.matrix.I)
	for i := 0; i < len(c.matrix.Indptr)-1; i++ {
		for j := c.matrix.Indptr[i]; j < c.matrix.Indptr[i+1]; j++ {
			dok.Set(c.matrix.Ind[j], i, c.matrix.Data[j])
		}
	}

	return dok
}

// ToCOO returns a COOrdinate sparse format version of the matrix.  The returned COO matrix will
// not share underlying storage with the receiver nor is the receiver modified by this call.
func (c *CSC) ToCOO() *COO {
	rows := make([]int, c.NNZ())
	cols := make([]int, c.NNZ())
	data := make([]float64, c.NNZ())

	for i := 0; i < len(c.matrix.Indptr)-1; i++ {
		for j := c.matrix.Indptr[i]; j < c.matrix.Indptr[i+1]; j++ {
			cols[j] = i
			rows[j] = c.matrix.Ind[j]
			data[j] = c.matrix.Data[j]
		}
	}

	coo := NewCOO(c.matrix.J, c.matrix.I, rows, cols, data)

	return coo
}

// ToCSR returns a Compressed Sparse Row sparse format version of the matrix.  The returned CSR matrix
// will not share underlying storage with the receiver nor is the receiver modified by this call.
// NB, the current implementation uses COO as an intermediate format so converts to COO before converting
// to CSR but attempts to reuse memory in the intermediate formats.
func (c *CSC) ToCSR() *CSR {
	return c.ToCOO().ToCSRReuseMem()
}

// ToCSC returns the receiver
func (c *CSC) ToCSC() *CSC {
	return c
}

// ToType returns an alternative format version fo the matrix in the format specified.
func (c *CSC) ToType(matType MatrixType) mat.Matrix {
	return matType.Convert(c)
}

// RowView slices the Compressed Sparse Column matrix along its secondary axis.
// Returns a VecDense dense Vector containing a copy of elements of row i.
func (c *CSC) RowView(i int) mat.Vector {
	if i >= c.matrix.J || i < 0 {
		panic(mat.ErrRowAccess)
	}
	slice := make([]float64, c.matrix.I)
	for j := range slice {
		slice[j] = c.At(i, j)
	}
	return mat.NewVecDense(c.matrix.I, slice)
}

// ColView slices the Compressed Sparse Column matrix along its primary axis.
// Returns a VecCOO sparse Vector that shares the same underlying storage as
// column i of the receiver.
func (c *CSC) ColView(j int) mat.Vector {
	if j >= c.matrix.I || j < 0 {
		panic(mat.ErrColAccess)
	}
	return NewVector(
		c.matrix.J,
		c.matrix.Ind[c.matrix.Indptr[j]:c.matrix.Indptr[j+1]],
		c.matrix.Data[c.matrix.Indptr[j]:c.matrix.Indptr[j+1]],
	)
}

// ScatterCol returns a slice representing column j of the matrix in dense format.  Col
// is used as the storage for the operation unless it is nil in which case, new
// storage of the correct length will be allocated.  This method will panic if j
// is out of range or col is not the same length as the number of rows in the matrix i.e.
// the correct size to receive the dense representation of the column.
func (c *CSC) ScatterCol(j int, col []float64) []float64 {
	if j >= c.matrix.I || j < 0 {
		panic(mat.ErrColAccess)
	}
	if col != nil && len(col) != c.matrix.J {
		panic(mat.ErrColLength)
	}
	if col == nil {
		col = make([]float64, c.matrix.J)
	}
	blas.Ussc(
		c.matrix.Data[c.matrix.Indptr[j]:c.matrix.Indptr[j+1]],
		col,
		1,
		c.matrix.Ind[c.matrix.Indptr[j]:c.matrix.Indptr[j+1]],
	)
	return col
}
