package usecase

// methodModules hold the questions of the methods of the first chapter
// (Вики/Я-концепция.docx, tables 15–19); they are asked only when the
// assignment includes the methods.
var methodModules = map[string]bool{"W": true, "U": true, "X": true, "K": true}

func methodModule(module string) bool { return methodModules[module] }
