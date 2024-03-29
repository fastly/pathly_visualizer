### `File Structure`
This section will show off the important portions in the front end's file structure
```javascript
. //Root folder ./frontend
├── node_modules //dependencies --> usually part of .gitignore
├── public //what's visible to the user
└── src
    ├── components //React functional components --> reuse code, ease up on readability
    |   ├── images //Images used to stylize components
    |   └── test_data //All test data used when testing ReactFlow library
    ├── App.js //Main page --> used as react router --> routes to links connected to components (i.e. "/" to ./components/Home.jsx)
    └── package.json //Includes important dependency downloads and scripts --> scripts and what they do can be seen below
``` 

### `npm install --legacy-peer-deps`
Installs all necessary node_modules required to run the project

### `npm start`

Runs the app in the development mode.\
Open [http://localhost:3000](http://localhost:3000) to view it in your browser.

The page will reload when you make changes.\
You may also see any lint errors in the console.

### `npm test`

Launches the test runner in the interactive watch mode.\
See the section about [running tests](https://facebook.github.io/create-react-app/docs/running-tests) for more information.

### `npm run build`

Builds the app for production to the `build` folder.\
It correctly bundles React in production mode and optimizes the build for the best performance.

The build is minified and the filenames include the hashes.\
Your app is ready to be deployed!

See the section about [deployment](https://facebook.github.io/create-react-app/docs/deployment) for more information.
