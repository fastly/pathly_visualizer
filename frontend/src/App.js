import './App.css';
import Test from './components/test'

function App() {

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
    // change to /api/post for testing w/o trcrt path specified yet
    xhr.open("POST", "http://localhost:8080/trcrt", true)
    // I'm assuming json is the format we want to be sending requests w/
    xhr.setRequestHeader("Content-Type", "application/json")
    // what happens when response is received
    xhr.onreadystatechange = () => {
      if(xhr.readyState === XMLHttpRequest.DONE && xhr.status === 200) {
        console.log(xhr.response)
        // TODO Send response to function to parse and create graph
        // graphs will most likely be their own component, might need to pass parsed response as props into component
      }
      else {
        console.error(xhr.response)
        alert("AN ERROR HAS OCCURRED: PLEASE TRY AGAIN")
      }
    }

    xhr.send(JSON.stringify(formObj))
  }

  return (
    <div className="App">
      {/* FORM FOR COLLECTING DATA FROM BACKEND STORAGE HERE */}
      <form id="postForm" onSubmit={search}>
        <input name="src" placeholder="Source Probe" required></input>
        {/* Using list of measurements sds suggested to start from */}
        <select name="dst" placeholder="Destination IP" required>
          {/* Separated using optgroup --> note I have no way to test this rn and I've never used optgroup before */}
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

        <button id="submitForm" type="submit">Submit</button>
      </form>
    </div>
  );
}

export default App;
