verify-links:
	@set -euEo pipefail; broken_links=( $$( find . -type l ! -exec test -e {} \; -print ) ); \
	if [[ -n "$${broken_links[@]}" ]]; then \
		echo "The following links are broken:" > /dev/stderr; \
		ls -l --color=auto $${broken_links[@]}; \
		exit 1; \
	fi;
.PHONY: verify-links
_default-verify-target: verify-links
