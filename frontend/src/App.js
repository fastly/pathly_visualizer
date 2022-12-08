import './App.css';
import { BrowserRouter as Router, Routes, 
  Route,} from "react-router-dom";
import Topbar from './components/Topbar';
import Home from './components/Home'
import About from './components/About';
import EditMeasurements from './components/EditMeasurements';


function App() {

  return (
    <>
      <Topbar></Topbar>
      <Router>
        <Routes>
          <Route exact path='/' element={<Home/>}></Route>
          <Route exact path='/about' element={<About/>}></Route>
          <Route exact path='/edit' element={<EditMeasurements/>}></Route>
        </Routes>
      </Router>
    </>
  );
}

export default App;
