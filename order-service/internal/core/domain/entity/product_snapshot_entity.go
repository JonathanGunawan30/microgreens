package entity

type ProductSnapshotEntity struct {
	ProductID int64
	Name      string
	Image     string
	SalePrice int64
	Unit      string
	Weight    int64
	IsActive  bool
}
