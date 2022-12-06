import Container from 'react-bootstrap/Container';
import Nav from 'react-bootstrap/Nav';
import Navbar from 'react-bootstrap/Navbar';

import 'bootstrap/dist/css/bootstrap.min.css';

function LogoDropdown() {

    return(
        <Navbar bg='light' expand='lg'>
            <Container>
                <Navbar.Brand><span style={{color: '#AA0000'}}>PATH</span>ly Visualizer</Navbar.Brand>
                <Navbar.Toggle aria-controls="basic-navbar-nav" />
                <Navbar.Collapse id="basic-navbar-nav">
                <Nav className="me-auto">
                    <Nav.Link href="">Edit Measurements</Nav.Link>
                    <Nav.Link href="https://atlas.ripe.net/">Ripe Atlas</Nav.Link>
                    <Nav.Link href="https://github.com/jmeggitt/fastly_anycast_experiments">Source Code</Nav.Link>
                </Nav>
                </Navbar.Collapse>
            </Container>
        </Navbar>
    )
}

export default LogoDropdown