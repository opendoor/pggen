package middleware_test

import (
	"context"
	"database/sql"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/opendoor-labs/pggen/middleware"
	"github.com/opendoor-labs/pggen/testing/mocks"
)

var _ = Describe("Middleware", func() {
	var (
		mockDBConn *mocks.DBConn
	)

	BeforeEach(func() {
		mockDBConn = &mocks.DBConn{}
	})

	Describe("ExecMiddleware", func() {
		expectedCtx := context.Background()
		expectedQuery := "myQuery"
		expectedArgs := []interface{}{1, "a"}
		expectedResult := &mocks.Result{}

		It("applies the middleware provided to the wrapper", func() {
			beforeExecDone := false
			afterExecDone := false
			testExecMiddleware := func(execFunc middleware.ExecFunc) middleware.ExecFunc {
				return func(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
					Expect(ctx).To(Equal(expectedCtx))
					Expect(query).To(Equal(expectedQuery))
					Expect(args).To(Equal(expectedArgs))
					beforeExecDone = true
					result, err := execFunc(ctx, query, args)
					Expect(result).To(Equal(expectedResult))
					Expect(err).NotTo(HaveOccurred())
					afterExecDone = true
					return result, err
				}
			}
			connWrapper := middleware.NewDBConnWrapper(mockDBConn).WithExecMiddleware(testExecMiddleware)

			mockDBConn.ExecContextReturns(expectedResult, nil)

			result, err := connWrapper.ExecContext(expectedCtx, expectedQuery, expectedArgs...)
			Expect(result).To(Equal(expectedResult))
			Expect(err).NotTo(HaveOccurred())

			Expect(beforeExecDone).To(BeTrue())
			Expect(afterExecDone).To(BeTrue())
		})
	})
})
