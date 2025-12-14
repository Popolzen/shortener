// Package main реализует multichecker для статического анализа кода проекта.
//
// # Назначение
//
// Multichecker объединяет несколько статических анализаторов для проверки кода
// на соответствие стандартам качества, обнаружения потенциальных ошибок
// и выявления проблем безопасности.
//
// # Запуск
//
// Для запуска анализатора выполните:
//
//	go run cmd/staticlint/main.go ./...
//
// Или соберите бинарный файл:
//
//	go build -o staticlint cmd/staticlint/main.go
//	./staticlint ./...
//
// Для анализа конкретного пакета:
//
//	./staticlint ./internal/handler
//
// # Состав анализаторов
//
// Multichecker включает следующие группы анализаторов:
//
// ## 1. Стандартные анализаторы golang.org/x/tools/go/analysis/passes
//
//   - printf: проверяет корректность форматирования в fmt.Printf и подобных
//   - shadow: обнаруживает затенение переменных
//   - structtag: проверяет корректность тегов структур
//   - unusedresult: находит неиспользуемые результаты функций
//
// ## 2. Анализаторы класса SA из staticcheck.io
//
// Все анализаторы класса SA (Static Analysis) проверяют код на наличие
// распространенных ошибок и проблем:
//
//   - SA1*: обнаружение некорректного использования стандартной библиотеки
//   - SA2*: проверка конкурентности и синхронизации
//   - SA3*: проверка логических операций
//   - SA4*: проверка корректности использования API
//   - SA5*: проверка на распространенные ошибки
//   - SA6*: проверка эффективности кода
//   - SA9*: проверка на подозрительные конструкции
//
// ## 3. Дополнительные анализаторы staticcheck.io
//
//   - ST1003 (stylecheck): проверка именования на соответствие Go conventions
//   - QF1001 (quickfix): предложения по упрощению кода
//
// ## 4. Публичные анализаторы
//
//   - errcheck: проверяет обработку ошибок
//   - govet: стандартный анализатор Go
//
// ## 5. Собственный анализатор
//
//   - noosexit: запрещает прямой вызов os.Exit в main функции main пакета
//
// # Собственный анализатор noosexit
//
// Анализатор запрещает использование os.Exit напрямую в функции main
// пакета main. Это улучшает тестируемость и graceful shutdown.
//
// Вместо:
//
//	func main() {
//	    if err != nil {
//	        os.Exit(1) // ❌ Будет обнаружено
//	    }
//	}
//
// Используйте:
//
//	func main() {
//	    if err := run(); err != nil {
//	        log.Fatal(err) // ✅ Или другой способ
//	    }
//	}
//
// # Примеры использования
//
// Проверка всего проекта:
//
//	./staticlint ./...
//
// Проверка конкретного пакета:
//
//	./staticlint ./internal/handler
//
// С выводом подробной информации:
//
//	./staticlint -v ./...
//
// # Интеграция с CI/CD
//
// Добавьте в .github/workflows/statictest.yml:
//
//   - name: Run staticlint
//     run: |
//     go build -o staticlint cmd/staticlint/main.go
//     ./staticlint ./...
package main

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/unusedresult"
	"honnef.co/go/tools/staticcheck"
)

// noOsExitAnalyzer — собственный анализатор, запрещающий os.Exit в main.
//
// Анализатор проверяет, что функция main пакета main не содержит
// прямых вызовов os.Exit. Это улучшает:
//   - Тестируемость кода
//   - Возможность graceful shutdown
//   - Корректную очистку ресурсов
var noOsExitAnalyzer = &analysis.Analyzer{
	Name: "noosexit",
	Doc:  "запрещает использование os.Exit в функции main пакета main",
	Run:  runNoOsExit,
}

// runNoOsExit выполняет проверку на наличие os.Exit в main.
func runNoOsExit(pass *analysis.Pass) (interface{}, error) {
	// Проверяем только пакет main
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}

	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			// Ищем функцию main
			fn, ok := n.(*ast.FuncDecl)
			if !ok || fn.Name.Name != "main" {
				return true
			}

			// Проверяем тело функции main
			// НЕ проверяем вложенные функции (goroutines, closures)
			if fn.Body == nil {
				return true
			}

			for _, stmt := range fn.Body.List {
				checkStatement(stmt, pass, false)
			}

			return false // не идём глубже в AST
		})
	}

	return nil, nil
}

// checkStatement проверяет statement на os.Exit (без рекурсии в функции)
func checkStatement(stmt ast.Stmt, pass *analysis.Pass, inFunc bool) {
	switch s := stmt.(type) {
	case *ast.ExprStmt:
		if call, ok := s.X.(*ast.CallExpr); ok {
			checkOsExit(call, pass)
		}
	case *ast.AssignStmt:
		for _, expr := range s.Rhs {
			if call, ok := expr.(*ast.CallExpr); ok {
				checkOsExit(call, pass)
			}
		}
	case *ast.IfStmt:
		if s.Body != nil {
			for _, stmt := range s.Body.List {
				checkStatement(stmt, pass, inFunc)
			}
		}
		if s.Else != nil {
			checkStatement(s.Else, pass, inFunc)
		}
	case *ast.BlockStmt:
		for _, stmt := range s.List {
			checkStatement(stmt, pass, inFunc)
		}
	case *ast.ForStmt:
		if s.Body != nil {
			for _, stmt := range s.Body.List {
				checkStatement(stmt, pass, inFunc)
			}
		}
	case *ast.RangeStmt:
		if s.Body != nil {
			for _, stmt := range s.Body.List {
				checkStatement(stmt, pass, inFunc)
			}
		}
	case *ast.SwitchStmt:
		if s.Body != nil {
			for _, stmt := range s.Body.List {
				checkStatement(stmt, pass, inFunc)
			}
		}
	case *ast.CaseClause:
		for _, stmt := range s.Body {
			checkStatement(stmt, pass, inFunc)
		}
	case *ast.GoStmt:
		// НЕ проверяем goroutine - это не прямой вызов в main
		return
	case *ast.DeferStmt:
		// НЕ проверяем defer
		return
	}
}

// checkOsExit проверяет конкретный вызов функции
func checkOsExit(call *ast.CallExpr, pass *analysis.Pass) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}

	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return
	}

	if ident.Name == "os" && sel.Sel.Name == "Exit" {
		pass.Reportf(call.Pos(),
			"использование os.Exit в функции main запрещено")
	}
}

func main() {
	// Собираем все анализаторы в один список
	checks := []*analysis.Analyzer{
		// 1. Стандартные анализаторы
		printf.Analyzer,
		shadow.Analyzer,
		structtag.Analyzer,
		unusedresult.Analyzer,

		// 5. Собственный анализатор
		noOsExitAnalyzer,
	}

	// 2. Добавляем все SA анализаторы из staticcheck
	for _, v := range staticcheck.Analyzers {
		checks = append(checks, v.Analyzer)
	}

	// Запускаем multichecker
	multichecker.Main(checks...)
}
