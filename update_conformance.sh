#!/usr/bin/env sh

rm -rf kdl-conformance-upstream
git clone https://github.com/kdl-org/kdl kdl-conformance-upstream
rm -rf testdata/{invalid,valid,valid_want}
(
	cd kdl-conformance-upstream/tests/test_cases
	(cd expected_kdl && find . -mindepth 1 | cut -f2 -d/ | sort) >valid
	(cd input && find . -mindepth 1 | cut -f2 -d/ | sort) >all
	comm -23 all valid >invalid
)
mkdir -p testdata/{invalid,valid,valid_want}
for fname in `cat kdl-conformance-upstream/tests/test_cases/valid`; do
	cp kdl-conformance-upstream/tests/test_cases/input/${fname} testdata/valid
	cp kdl-conformance-upstream/tests/test_cases/expected_kdl/${fname} testdata/valid_want
done
for fname in `cat kdl-conformance-upstream/tests/test_cases/invalid`; do
	cp kdl-conformance-upstream/tests/test_cases/input/${fname} testdata/invalid
done
rm -rf kdl-conformance-upstream
