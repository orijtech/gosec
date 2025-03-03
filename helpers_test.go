package gosec_test

import (
	"go/ast"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	"github.com/orijtech/gosec/v2"
	"github.com/orijtech/gosec/v2/testutils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Helpers", func() {
	Context("when listing pacakge paths", func() {
		var dir string
		JustBeforeEach(func() {
			var err error
			dir, err = ioutil.TempDir("", "gosec")
			Expect(err).ShouldNot(HaveOccurred())
			_, err = ioutil.TempFile(dir, "test*.go")
			Expect(err).ShouldNot(HaveOccurred())
		})
		AfterEach(func() {
			err := os.RemoveAll(dir)
			Expect(err).ShouldNot(HaveOccurred())
		})
		It("should return the root directory as package path", func() {
			paths, err := gosec.PackagePaths(dir, nil)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(paths).Should(Equal([]string{dir}))
		})
		It("should return the package package path", func() {
			paths, err := gosec.PackagePaths(dir+"/...", nil)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(paths).Should(Equal([]string{dir}))
		})
		It("should exclude folder", func() {
			nested := dir + "/vendor"
			err := os.Mkdir(nested, 0755)
			Expect(err).ShouldNot(HaveOccurred())
			_, err = os.Create(nested + "/test.go")
			Expect(err).ShouldNot(HaveOccurred())
			exclude, err := regexp.Compile(`([\\/])?vendor([\\/])?`)
			Expect(err).ShouldNot(HaveOccurred())
			paths, err := gosec.PackagePaths(dir+"/...", []*regexp.Regexp{exclude})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(paths).Should(Equal([]string{dir}))
		})
		It("should be empty when folder does not exist", func() {
			nested := dir + "/test"
			paths, err := gosec.PackagePaths(nested+"/...", nil)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(paths).Should(BeEmpty())
		})
	})

	Context("when getting the root path", func() {
		It("should return the absolute path from relative path", func() {
			base := "test"
			cwd, err := os.Getwd()
			Expect(err).ShouldNot(HaveOccurred())
			root, err := gosec.RootPath(base)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(root).Should(Equal(filepath.Join(cwd, base)))
		})
		It("should retrun the absolute path from ellipsis path", func() {
			base := "test"
			cwd, err := os.Getwd()
			Expect(err).ShouldNot(HaveOccurred())
			root, err := gosec.RootPath(filepath.Join(base, "..."))
			Expect(err).ShouldNot(HaveOccurred())
			Expect(root).Should(Equal(filepath.Join(cwd, base)))
		})
	})

	Context("when excluding the dirs", func() {
		It("should create a proper regexp", func() {
			r := gosec.ExcludedDirsRegExp([]string{"test"})
			Expect(len(r)).Should(Equal(1))
			match := r[0].MatchString("/home/go/src/project/test/pkg")
			Expect(match).Should(BeTrue())
			match = r[0].MatchString("/home/go/src/project/vendor/pkg")
			Expect(match).Should(BeFalse())
		})

		It("should create no regexp when dir list is empty", func() {
			r := gosec.ExcludedDirsRegExp(nil)
			Expect(len(r)).Should(Equal(0))
			r = gosec.ExcludedDirsRegExp([]string{})
			Expect(len(r)).Should(Equal(0))
		})
	})

	Context("when getting call info", func() {
		It("should return the type and call name for selector expression", func() {
			pkg := testutils.NewTestPackage()
			defer pkg.Close()
			pkg.AddFile("main.go", `
			package main

			import(
			    "bytes"
			)

			func main() {
			    b := new(bytes.Buffer)
				_, err := b.WriteString("test")
				if err != nil {
				    panic(err)
				}
			}
			`)
			ctx := pkg.CreateContext("main.go")
			result := map[string]string{}
			visitor := testutils.NewMockVisitor()
			visitor.Context = ctx
			visitor.Callback = func(n ast.Node, ctx *gosec.Context) bool {
				typeName, call, err := gosec.GetCallInfo(n, ctx)
				if err == nil {
					result[typeName] = call
				}
				return true
			}
			ast.Walk(visitor, ctx.Root)

			Expect(result).Should(HaveKeyWithValue("*bytes.Buffer", "WriteString"))
		})

		It("should return the type and call name for new selector expression", func() {
			pkg := testutils.NewTestPackage()
			defer pkg.Close()
			pkg.AddFile("main.go", `
			package main

			import(
			    "bytes"
			)

			func main() {
				_, err := new(bytes.Buffer).WriteString("test")
				if err != nil {
				    panic(err)
				}
			}
			`)
			ctx := pkg.CreateContext("main.go")
			result := map[string]string{}
			visitor := testutils.NewMockVisitor()
			visitor.Context = ctx
			visitor.Callback = func(n ast.Node, ctx *gosec.Context) bool {
				typeName, call, err := gosec.GetCallInfo(n, ctx)
				if err == nil {
					result[typeName] = call
				}
				return true
			}
			ast.Walk(visitor, ctx.Root)

			Expect(result).Should(HaveKeyWithValue("bytes.Buffer", "WriteString"))
		})

		It("should return the type and call name for function selector expression", func() {
			pkg := testutils.NewTestPackage()
			defer pkg.Close()
			pkg.AddFile("main.go", `
			package main

			import(
			    "bytes"
			)

			func createBuffer() *bytes.Buffer {
			    return new(bytes.Buffer)
			}

			func main() {
				_, err := createBuffer().WriteString("test")
				if err != nil {
				    panic(err)
				}
			}
			`)
			ctx := pkg.CreateContext("main.go")
			result := map[string]string{}
			visitor := testutils.NewMockVisitor()
			visitor.Context = ctx
			visitor.Callback = func(n ast.Node, ctx *gosec.Context) bool {
				typeName, call, err := gosec.GetCallInfo(n, ctx)
				if err == nil {
					result[typeName] = call
				}
				return true
			}
			ast.Walk(visitor, ctx.Root)

			Expect(result).Should(HaveKeyWithValue("*bytes.Buffer", "WriteString"))
		})

		It("should return the type and call name for package function", func() {
			pkg := testutils.NewTestPackage()
			defer pkg.Close()
			pkg.AddFile("main.go", `
			package main

			import(
			    "fmt"
			)

			func main() {
			    fmt.Println("test")
			}
			`)
			ctx := pkg.CreateContext("main.go")
			result := map[string]string{}
			visitor := testutils.NewMockVisitor()
			visitor.Context = ctx
			visitor.Callback = func(n ast.Node, ctx *gosec.Context) bool {
				typeName, call, err := gosec.GetCallInfo(n, ctx)
				if err == nil {
					result[typeName] = call
				}
				return true
			}
			ast.Walk(visitor, ctx.Root)

			Expect(result).Should(HaveKeyWithValue("fmt", "Println"))
		})
	})
	Context("when getting binary expression operands", func() {
		It("should return all operands of a binary experssion", func() {
			pkg := testutils.NewTestPackage()
			defer pkg.Close()
			pkg.AddFile("main.go", `
			package main

			import(
			    "fmt"
			)

			func main() {
				be := "test1" + "test2"
				fmt.Println(be)
			}
			`)
			ctx := pkg.CreateContext("main.go")
			var be *ast.BinaryExpr
			visitor := testutils.NewMockVisitor()
			visitor.Context = ctx
			visitor.Callback = func(n ast.Node, ctx *gosec.Context) bool {
				if expr, ok := n.(*ast.BinaryExpr); ok {
					be = expr
				}
				return true
			}
			ast.Walk(visitor, ctx.Root)

			operands := gosec.GetBinaryExprOperands(be)
			Expect(len(operands)).Should(Equal(2))
		})
		It("should return all operands of complex binary experssion", func() {
			pkg := testutils.NewTestPackage()
			defer pkg.Close()
			pkg.AddFile("main.go", `
			package main

			import(
			    "fmt"
			)

			func main() {
				be := "test1" + "test2" + "test3" + "test4"
				fmt.Println(be)
			}
			`)
			ctx := pkg.CreateContext("main.go")
			var be *ast.BinaryExpr
			visitor := testutils.NewMockVisitor()
			visitor.Context = ctx
			visitor.Callback = func(n ast.Node, ctx *gosec.Context) bool {
				if expr, ok := n.(*ast.BinaryExpr); ok {
					if be == nil {
						be = expr
					}
				}
				return true
			}
			ast.Walk(visitor, ctx.Root)

			operands := gosec.GetBinaryExprOperands(be)
			Expect(len(operands)).Should(Equal(4))
		})
	})
})
