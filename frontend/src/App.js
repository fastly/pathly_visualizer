import { useState } from 'react';
import './App.css';
import Graph from './components/Graph';

function App() {

  // list of graphs being rendered --> start with state []
  const [graphList, setGraphList] = useState([])

  // Search function --> connect to rest API
  // Once RIS data collection is implemented, this will include that as well
  const search = (e) => {
    e.preventDefault()

    // get serialized form data
    var form = new FormData(e.target)

    // convert form data into object to send in http rq
    const formObj = {}
    form.forEach((value, key) => (formObj[key] = value))

    // create new http rq --> note, I've used XMLHttpRequest before but if there's a preferred method of sending requests use that instead
    const xhr = new XMLHttpRequest()
    // I assume keeping localhost here is fine as the code will be running on GCP regardless
    xhr.open("POST", "http://localhost:8080/api/traceroute/clean", true)
    xhr.setRequestHeader("Content-Type", "application/json")
    // what happens when response is received
    xhr.onreadystatechange = () => {
      if(xhr.readyState === XMLHttpRequest.DONE && xhr.status === 200) {
        console.log(xhr.response)
        //concat graph onto current graph list, gets rerendered w/ new graph list
        setGraphList(graphList.concat(<Graph response={xhr.response}></Graph>))
      }
    }

    xhr.send(JSON.stringify(formObj))
  }

  return (
    <>
      <div className="App">
        {/* FORM FOR COLLECTING DATA FROM BACKEND STORAGE HERE */}
        <h1>CREATE VISUALIZATION</h1>
        <form id="postForm" onSubmit={search}>
        <label for="src">Source Probe</label>
          <input id= "srcProbe" name="src" placeholder="e.g. 123456" required></input>
          
          <br></br>
          {/* Using list of measurements sds suggested to start from */}
          <label for="dst">Destination IP</label>
          <select id="destIP" name="dst" placeholder="Destination IP" required>
          {/* <option selected="true" style={{display: 'none'}}></option> */}
          <option hidden> Select IP Address</option>
            <optgroup label="k-root">
              <option value="193.0.14.129">193.0.14.129</option>
              <option value="2001:7fd::1">2001:7fd::1</option>
            </optgroup>
            <optgroup label="b-root">
              <option value="199.9.14.201">199.9.14.201</option>
              <option value="2001:500:200::b">2001:500:200::b</option>
            </optgroup>
            <optgroup label="fastly anycast">
              <option value="151.101.0.1">151.101.0.1</option>
              <option value="2a04:4e42::1">2a04:4e42::1</option>
            </optgroup>
          </select>
          <br></br>
          <button id="submitForm" type="submit">Visualize</button>
        </form>
      </div>
      {/* Left empty, graphs rendered on response load */}
      <button onClick={addGraph}>Add</button>
      <div id="graphArea" style={{height: 600, width: 600}}>
        {graphList}
      </div>
    </>
  );
}

export default App;
