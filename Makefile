new-book:
	npx zenn new:book --slug ${slug}
	cd books/${slug} && mv example1.md intro.md && mv example2.md appendix.md
	sed -i '' -e 's/example1/intro/g' books/${slug}/config.yaml
	sed -i '' -e 's/example2/appendix/g' books/${slug}/config.yaml

new-page:
	echo "---\ntitle: \"\"\n---" > books/${book}/${slug}.md

preview:
	npx zenn preview