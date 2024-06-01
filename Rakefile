require "rake/clean"
require "fileutils"

CLOBBER.include("bin/web-starter-app")

file "bin/web-starter-app" => FileList["Rakefile", "*.go", "go.*", "**/*.go"].exclude(/_test.go$/) do |t|
  args = ["go", "build", "-o", t.name]

  # Uncomment the following line to enable debugging.
  # args << "-gcflags" << "all=-N -l"

  sh *args
end

desc "Build"
task build: ["bin/web-starter-app"]

desc "Run web-starter-app"
task run: :build do
  exec "bin/web-starter-app serve"
end

desc "Watch for source changes and rebuild and rerun"
task :rerun do
  exec %q[watchexec -r -f Rakefile -f "**/*.go" --ignore "**/*_test.go" -- rake run]
end
