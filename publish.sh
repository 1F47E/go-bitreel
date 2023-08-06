git fetch --tags
latest_tag=$(git tag -l | grep "$1" | sort -V | tail -n 1 | cut -d' ' -f1)

while true; do
  new_tag=$(echo $latest_tag | awk -F. '{$NF+=1} 1' OFS=.)
  if ! git rev-parse "$new_tag" >/dev/null 2>&1; then
    break
  else
    latest_tag=$new_tag
  fi
done

echo "Pushing new tag: $new_tag"
git push
git tag -a $new_tag -m "release $new_tag"
git push --tags
echo "Building release"
rm -rf ./dist
#goreleaser release --snapshot --clean
# goreleaser release
#git push origin $new_tag
