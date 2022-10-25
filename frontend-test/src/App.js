import './App.css';

function App() {

  const getRq = (e) => {
    //prevent form from reloading page on submission
    e.preventDefault()

    //create http request to send to endpoint
    const xhr = new XMLHttpRequest()
    xhr.open("GET", "http://localhost:8080/api/hello", true)

    xhr.setRequestHeader("Content-Type", "application/json")
    //on response
    xhr.onreadystatechange = () => {
      if(xhr.readyState === XMLHttpRequest.DONE && xhr.status === 200) {
        console.log(xhr.response)
      }
    }

    //send request (no body b/c it's a GET rq)
    xhr.send()
  }

  const postRq = (e) => {
    //prevent form from reloading page on submission
    e.preventDefault()

    //create new form data based on form
    var form = new FormData(e.target)

    //create json object of form data
    const formObj = {};
    form.forEach((value, key) => (formObj[key] = value))

    //create http request to send to endpoint
    const xhr = new XMLHttpRequest()
    xhr.open("POST", "http://localhost:8080/api/hello", true)

    xhr.setRequestHeader("Content-Type", "application/json")
    //on response 
    xhr.onreadystatechange = () => {
      if(xhr.readyState === XMLHttpRequest.DONE && xhr.status === 200) {
        console.log(xhr.response)
      }
    }
    //send form data
    xhr.send(JSON.stringify(formObj))
  }  

  return (
    <div className="App">
      <button type="button" onClick={getRq}>GET</button>
      <form onSubmit={postRq}>
        <input name="iTest" placeholder="test"></input>
        <br/>
        <button type="submit">POST</button>
      </form>
    </div>
  );
}

export default App;
