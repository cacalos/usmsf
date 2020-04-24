package configmgr

type GetData interface {
	GetDecisionConfig() error
	GetCommonConfig() error
	GetSmscConfig() error
}
