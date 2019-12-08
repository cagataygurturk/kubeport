const listNamespacesCommand = {"Type": "listNamespacesCommand"};
const listActiveConnectionsCommand = {"Type": "listActiveConnectionsCommand"};

const hostName = "com.cagataygurturk.kubeport";
let port = null;

const messaging = {
    connect: function (hostname, handler) {
        port = chrome.runtime.connectNative(hostName);
        port.onMessage.addListener(handler);
        port.onDisconnect.addListener(this.onDisconnected);
    },
    onDisconnected: function () {
        console.log("Failed to connect: " + chrome.runtime.lastError.message);
        port = null;
    },
    sendNativeMessage: function (message) {
        //message = JSON.parse(document.getElementById('input-text').value);
        port.postMessage(message);
        console.log("Sent message: " + JSON.stringify(message));
    }
};


$(document).ready(function () {

    const activeConnectionsLoading = function (isLoading) {
        const cls = 'active-connections-loading';
        if (isLoading) {
            $('body').addClass(cls);
        } else {
            $('body').removeClass(cls);
        }
    };

    const newConnectionsLoading = function (isLoading) {
        const cls = 'new-connections-loading';
        if (isLoading) {
            $('body').addClass(cls);
        } else {
            $('body').removeClass(cls);
        }
    };

    const getActiveConnections = function () {
        activeConnectionsLoading(true);
        messaging.sendNativeMessage(listActiveConnectionsCommand);
    };

    messaging.connect(hostName, function (message) {
        console.log(message);
        switch (message.Type) {
            case "listNamespacesResponse":
                const namespacesInput = $("#namespaces");
                namespacesInput.empty();
                $.each(message.Payload, function (i, item) {
                    namespacesInput.append($("<option />").val(item.metadata.name).text(item.metadata.name));
                });
                namespacesInput.off().on('change', function () {
                    const selectedNamespace = $(this).val();
                    newConnectionsLoading(true);
                    messaging.sendNativeMessage({"Type": "listServicesCommand", "Payload": selectedNamespace})
                });
                newConnectionsLoading(false);
                getActiveConnections();
                break;
            case "listServicesResponse":
                const servicesInput = $("#services");
                servicesInput.empty();
                $.each(message.Payload, function (i, item) {
                    console.log(item)
                    $.each(item.spec.ports, function (j, port) {
                        const val = item.metadata.name + ":" + port.port;
                        servicesInput.append($("<option />").val(val).text(val));
                    });
                });
                if (message.Payload.length === 0) {
                    servicesInput.append('<option disabled selected value="">No service found</option>');
                    $('#submit').hide();
                } else {
                    $('#submit').show();
                }
                newConnectionsLoading(false);
                getActiveConnections();
                break;


            case "connectServiceCommandResponse":
                //Give some time to kubectl to stabilize
                window.setTimeout(function () {
                    chrome.tabs.create({url: "localhost:" + message.Payload.localPort});
                }, 1000);

                getActiveConnections();
                break;

            case "listActiveConnectionsResponse":

                const table = $('#activeConnections');
                table.empty();
                table.append('<tr><td>Namespace</td><td>Service</td><td>Local Port</td><td></td></tr>')
                $.each(message.Payload, function (i, item) {
                    const link = $('<a href="#">' + item.service + ':' + item.remotePort + '</a>').on('click', function (e) {
                        chrome.tabs.create({url: "localhost:" + item.localPort});
                        e.preventDefault();
                    });

                    const row = $('<tr />');
                    row.append($('<td>' + item.namespace + '</td>'));

                    const column = $('<td />');
                    link.appendTo(column);
                    column.appendTo(row);

                    row.append('<td>' + item.localPort + '</td>');

                    const killColumn = $('<td />');
                    $('<a href="#">kill</a>').on('click', function (e) {
                        $(this).addClass("loading-kill-operation").html('&nbsp;');
                        messaging.sendNativeMessage({"Type": "killConnectionCommand", "Payload": parseInt(item.pid)});
                        e.preventDefault()
                    }).appendTo(killColumn);
                    killColumn.appendTo(row);

                    row.appendTo(table);

                });

                activeConnectionsLoading(false);
                //window.setTimeout(getActiveConnections, 10000);
                break;

        }
    });

    $('form').submit(function (event) {
        const selectedService = {};
        $.each($(this).serializeArray(), function (j, item) {
            selectedService[item.name] = item.value;
        });

        const serviceNameParsed = selectedService.service.split(":");
        selectedService.service = serviceNameParsed[0];
        selectedService.port = serviceNameParsed[1];

        messaging.sendNativeMessage({"Type": "connectServiceCommand", "Payload": selectedService});

        event.preventDefault();
    });

    // Start the action with listing namespaces
    newConnectionsLoading(true);
    messaging.sendNativeMessage(listNamespacesCommand);

});