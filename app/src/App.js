import React, {Component} from "react";
import "./App.css";
import Message from "./Message";
import axios from "axios";
import rock from "./rock.jpg";

class App extends Component {
    constructor() {
        super();

        this.state = {
            hasStarted: false,
            username: '',
            goldToBet: 1,
            messages: [],
            gold: 100
        }

    }

    // Initialize the web socket to allow for sending the throw message.
    initializeWebSocket() {

        this.socket = new WebSocket('ws://localhost:8000/challenge');

        this.socket.addEventListener('message', (event) => {
            const data = JSON.parse(event.data);
            if (data.error) {
                this.setState({
                    messages: this.state.messages.concat(data.error)
                });
            } else {
                // TODO/NOTE: a player could easily only update the token when they win, without nonce, this is open to replay attacks
                this.token = data.token;
                let messages = this.state.messages;
                let gold = this.state.gold;
                if (data.outcome === 'WIN') {
                    let message = `You won ${data.gold - gold} gold from ${data.opposer}!`
                    if (this.state.currentThrow === 'r') {
                        message += ' They smell what\'s cookin\'.'
                    }
                    messages = messages.concat(message);
                } else if (data.outcome === 'LOSS') {
                    messages = messages.concat(`You lost ${gold - data.gold} gold to ${data.opposer}...`);
                } else if (data.outcome === 'TIE') {
                    messages = messages.concat(`You tied with ${data.opposer}.`);
                } else {
                    messages = messages.concat(`Unexpected outcome ${data.outcome}`);
                }
                this.setState({
                    gold: data.gold,
                    messages
                });

                this.initializeWebSocket();
            }
        });

    }

    componentWillUnmount() {
        this.socket.close();
    }

    // Send a message with the current bet and throw
    sendMessage(throwType) {
        if (this.state.goldToBet < 1) {
            this.setState({
                messages: this.state.messages.concat('You can\'t bet less than 1 gold...')
            });
            return;
        }
        this.setState({
            currentThrow: throwType
        });
        this.socket.send(JSON.stringify({
            token: this.token,
            'throw': throwType,
            gold: this.state.goldToBet
        }));
    }

    handleUsernameUpdate(event) {
        this.setState({username: event.target.value});
    }

    handleGoldToBetUpdate(event) {
        try{
            this.setState({goldToBet: parseInt(event.target.value)});
        } catch (e) {
            this.setState({messages: this.state.messages.concat('Failed to parse int from ' + event.target.value)})
        }
    }

    // Register the user by a username when they first enter.
    // The returned token represents the signed user state, keep it to
    // send with ws messages for validation.
    handleRegister() {
        axios.post('http://localhost:8000/register', {
            username: this.state.username
        })
            .then((response) => {
                this.token = response.data;
                this.initializeWebSocket();
                this.setState({
                    hasStarted: true,
                    messages: []
                });
            }, (error) => {
                console.error(error);
                this.setState({
                    messages: this.state.messages.concat('Error in registering user (May need to provide valid username?).')
                });
            });
    }

    render() {
        return (
            <div className="App">
                {
                    !this.state.hasStarted ? <div>
                        Username: <input type="text" value={this.state.username}
                                         onChange={this.handleUsernameUpdate.bind(this)}/>
                        <button onClick={this.handleRegister.bind(this)}>Register</button>
                    </div> :
                        <div>
                            <div>Current gold: {this.state.gold}</div>
                            <div>
                                Gold to bet: <input type="number" min="1" value={this.state.goldToBet}
                                                    onChange={this.handleGoldToBetUpdate.bind(this)}/>
                            </div>
                            <button onClick={this.sendMessage.bind(this, 'r')} className="throw-button rock-button">
                                <img src={rock}
                                     alt="Do you smell what the Rock is cookin'?"/>
                                <span>Rock</span>
                            </button>
                            <button onClick={this.sendMessage.bind(this, 'p')} className="throw-button">Paper</button>
                            <button onClick={this.sendMessage.bind(this, 's')} className="throw-button">Scissors
                            </button>
                        </div>
                }

                {
                    this.state.messages.map((message, index) => <Message key={index}
                                                                         message={message}></Message>)
                }


            </div>
        );
    }
}

export default App;
