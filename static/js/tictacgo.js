var connection;

var canMove = false;

function getCellId(row, column) {
	return "cell_" + row + "_" + column;
}

function clearBoard() {
	$(".cell").each(function() {
		$(this).removeClass("busy").text("");
	});
}

function highlightCells(cells) {
	cells.forEach(function(item, index, cells){
		var row = item.Row;
		var column = item.Column;
		$("#" + getCellId(row, column)).addClass("win");
	});
}

function updateBoard(cells) {
	console.log(cells);
	clearBoard();
	cells.forEach(function(item, index, cells){
		var row = item.Row;
		var column = item.Column;
		$("#" + getCellId(row, column))
			.text(String.fromCharCode(item.Symbol))
			.addClass("busy");
	});
}

function makeWebSocketConnection( host ) {
	if( window["WebSocket"] ) {
		connection = new WebSocket( "ws://" + host + "/websocket" );
		connection.onopen = function() {
			updateStatusMessage( "You connected to the server." );
		};
		connection.onclose = function(event) {
			updateStatusMessage( "You has been disconnected." );
		};
		connection.onmessage = function(event) {
			// Проверяем, какое сообщение пришло.
			console.log(event.data);
			var data = JSON.parse(event.data)
			if( "Board" in data ) {
				updateBoard(data.Board.Cells);
				$("#currentPlayers").text( data.ActivePlayers );
				// Проверяем очерёдность.
				canMove = data.CanMove;
				if( data.CanMove ) {
					updateStatusMessage( "Your move!" );
				} else {
					updateStatusMessage( "Please, wait your move..." );
				}
				// Проверяем, вдруг кто-то выиграл партию
				if( data.Board.Winner != 0 ) {
					updateStatusMessage( "Player " + String.fromCharCode( data.Board.Winner ) + "won the game!" );
				}
			}
		}
	} else {
		updateStatusMessage( "Sorry, but your browser doesn\'t support WebSocket :(" )
	}
}

function updateStatusMessage( msg ) {
	$("#gameStatus").text( msg );
	console.log( msg );
}

function generateField( squareSize ) {
	$("#field").empty();
	for( var rowIndex = 0; rowIndex < squareSize; rowIndex++ ) {
		var row = document.createElement( "div" );
		$( "#field" ).append( row );
		$(row).addClass( "row" );
		for( var columnIndex = 0; columnIndex < squareSize; columnIndex++ ) {
			var cell = document.createElement( "div" );
			$(cell).addClass( "cell" );
			$(cell).attr("id", getCellId(rowIndex, columnIndex));
			$(cell).data("row-index", rowIndex).data("column-index", columnIndex);
			$(row).append( cell );
		}
	}
}

function enableCells() {
	$(".cell").click( function() {
		console.log("cell_click");
		if( !canMove ) {
			return;
		}
		var rowIndex = $(this).data("row-index");
		var columnIndex = $(this).data("column-index");
		// Проверяем, не занята ли клетка.
		if( $("#" + getCellId(rowIndex, columnIndex)).hasClass("busy") ) {
			updateStatusMessage("Cell you clicked is busy!");
			return;
		}
		// Отправляем сообщение на сервер.
		connection.send(JSON.stringify({
			Row: rowIndex,
			Column: columnIndex
		}));
		updateStatusMessage( "User clicked: (" + rowIndex + ", " + columnIndex + ")" );
	} );
}