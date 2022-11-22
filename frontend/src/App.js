import { useState } from 'react';
import './App.css';
import Graph from './components/Graph';

//test data for graph population
import {tData} from './components/testTrcrt'
import {tDataFull} from './components/testTrcrtFull'

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

    // split into two form obj to make two http requests
    let ipSplit = formObj.destinationIp.split(" / ")
    const ipv4FormObj = 
      {
        probeId: parseInt(formObj.probeId),
        destinationIp: ipSplit[0],
      }
    const ipv6FormObj = 
      {
        probeId: parseInt(formObj.probeId),
        destinationIp: ipSplit[1],
      }

    const sendingObjs = [ipv4FormObj, ipv6FormObj] 

    // create new http rq --> note, I've used XMLHttpRequest before but if there's a preferred method of sending requests use that instead
    const xhr = new XMLHttpRequest();
    
    let combProbeIp = "";
    let combNodes = [];
    let combEdges = [];
    // need to loop to synchronously make requests for ipv4 and 6
    (function loop(i, length) {
      if(i >= length){
        return
      }
      // I assume keeping localhost here is fine as the code will be running on GCP regardless
      if(document.getElementById("fullOrClean").checked){
        xhr.open("POST", "http://localhost:8080/api/traceroute/clean", true)
      }
      else{
        xhr.open("POST", "http://localhost:8080/api/traceroute/full", true)
      }
      xhr.setRequestHeader("Content-Type", "application/json")
      // what happens when response is received
      xhr.onreadystatechange = () => {
        if(xhr.readyState === XMLHttpRequest.DONE && xhr.status === 200) {
          let res = JSON.parse(xhr.response)
          console.log(res)
          // if request was made for ipv6, combine ipv4 and 6 responses and concat graph onto graphlist
          combNodes = combNodes.concat(res.nodes)
          combEdges = combEdges.concat(res.edges)
          if(combProbeIp === ""){
            combProbeIp = res.probeIp
          }
          else{
            combProbeIp = combProbeIp + " / " + res.probeIp
          }

          if(i === 1){
            let combinedResponse = {
              probeIp: combProbeIp,
              nodes: combNodes,
              edges: combEdges,
            }
            console.log(combinedResponse)
            //concat graph onto current graph list, gets rerendered w/ new graph list
            setGraphList(graphList.concat(<Graph response={combinedResponse} form={formObj} clean={document.getElementById("fullOrClean").checked}></Graph>))
          }
          loop(i+1, length)
        }
      }

      console.log("sending request")
      console.log(sendingObjs[i])
      //two requests for ipv4 and 6 addresses
      xhr.send(JSON.stringify(sendingObjs[i]))
    })(0, sendingObjs.length)
    
  }



  return (
    <>
      <div className="App">
        {/* FORM FOR COLLECTING DATA FROM BACKEND STORAGE HERE */}
        <h1>CREATE VISUALIZATION</h1>
        <form id="postForm" onSubmit={search}>
        <label for="src">Source Probe</label>
          <input id= "srcProbe" name="probeId" placeholder="e.g. 123456" required></input>
          <br></br>
          {/* Using list of measurements sds suggested to start from */}
          <label for="dst">Destination IP</label>
          <select id="destIP" name="destinationIp" placeholder="Destination IP" required>
          <option hidden> Select IP Address</option>
            {/* <optgroup label="k-root">
              <option value="193.0.14.129 / 2001:7fd::1">193.0.14.129 / 2001:7fd::1</option>
            </optgroup>
            <optgroup label="b-root">
              <option value="199.9.14.201 / 2001:500:200::b">199.9.14.201 / 2001:500:200::b</option>
            </optgroup> */}
            <optgroup label="fastly anycast">
              <option value="151.101.0.1 / 2a04:4e42::1">151.101.0.1 / 2a04:4e42::1</option>
            </optgroup>
          </select>
          <br></br>
          <div className="switchBox">
            <p>Full Data</p>
            <label className="switch">
              <input type="checkbox" name="fullOrClean" id="fullOrClean"/>
              <span className="slider round"></span>
            </label>
            <p>Clean Data</p>
          </div>
          <br></br>
          <button id="submitForm" type="submit">Visualize</button>
        </form>
      </div>
      {/* Left empty, graphs rendered on response load */}
      <div id="graphArea">
        {graphList}
        {/* <Graph response={tDataFull} clean={false}></Graph> */}
        {/* <Graph response={tData} clean={true}></Graph> */}
      </div>
    </>
  );
}

export default App;
