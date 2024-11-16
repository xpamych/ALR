package gen

import (
    "strings"
    "text/template"
)

// Определяем переменную funcs типа template.FuncMap, которая будет использоваться для
// предоставления пользовательских функций в шаблонах
var funcs = template.FuncMap{
    // Функция "tolower" использует strings.ToLower
    // для преобразования строки в нижний регистр
    "tolower": strings.ToLower,

    // Функция "firstchar" — это лямбда-функция, которая берет строку
    // и возвращает её первый символ
    "firstchar": func(s string) string {
        return s[:1]
    },
}
