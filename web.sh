rm -rf public

git clone https://github.com/maxischmaxi/league-game-web web
cd web
npm install
VITE_API_GATEWAY="" && npm run build

echo "Copying files to public folder"
cp -r ./dist ../public

cd ..
rm -rf web
echo "Done"
