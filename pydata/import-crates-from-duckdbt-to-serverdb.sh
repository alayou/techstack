go build -o techstacktools.exe ../cmd/techstacktools
./techstacktools.exe import  cargo  --source duckdb --config ../config.yml crates_io.duckdb 