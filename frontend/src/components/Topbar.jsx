import Container from 'react-bootstrap/Container';
import Nav from 'react-bootstrap/Nav';
import Navbar from 'react-bootstrap/Navbar';

import 'bootstrap/dist/css/bootstrap.min.css';

// top navbar component

function Topbar() {

    return(
        <Navbar bg='light' expand='lg' style={{zIndex: 1}}>
            <Container>
                <Navbar.Brand href="/"><img src={require("./images/pathly_logo.png")} width={120}></img></Navbar.Brand>
                <Navbar.Toggle aria-controls="basic-navbar-nav" />
                <Navbar.Collapse id="basic-navbar-nav">
                <Nav className="me-auto">
                    <Nav.Link href="/edit">Edit Measurements</Nav.Link>
                    <Nav.Link href="https://atlas.ripe.net/">Ripe Atlas</Nav.Link>
                    <Nav.Link href="https://github.com/jmeggitt/fastly_anycast_experiments">Source Code</Nav.Link>
                    <Nav.Link href="/about">About</Nav.Link>
                </Nav>
                </Navbar.Collapse>
            </Container>
        </Navbar>
    )
}

export default Topbar