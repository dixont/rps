import React, {Component} from "react";
import "./App.css";
import Message from './Message';

class App extends Component {
    constructor() {
        super();

        this.state = {
            messages: []
        }

    }

    componentDidMount() {
        this.socket = new WebSocket('ws://localhost:8000/lobby');

        this.socket.addEventListener('open', (event) => {
            console.log('Connected to lobby');
        });

        this.socket.addEventListener('message', (event) => {
            this.setState({
                messages: this.state.messages.concat(event.data)
            });
        });
    }

    componentWillUnmount() {
        this.socket.close();
    }

    sendMessage() {
        this.socket.send('Test message!');
    }

    render() {
        return (
            <div className="App">
                <button onClick={this.sendMessage.bind(this)}>Ping</button>
                {
                    this.state.messages.map(message => <Message message={message}></Message>)
                }
            </div>
        );
    }
}

export default App;
