import QtQuick
import QtQuick.Layouts
import QtQuick.Controls as QQC2
import org.kde.plasma.plasmoid
import org.kde.plasma.core as PlasmaCore
import org.kde.plasma.components as PlasmaComponents3
import org.kde.plasma.extras as PlasmaExtras
import org.kde.plasma.plasma5support as P5Support
import org.kde.kirigami as Kirigami

PlasmoidItem {
    id: root
    property string droboUsable: "10.93 TB"
    property string droboFree: "8.13 TB"
    property string droboHealth: ""

    Plasmoid.icon: hasDrobo() ? "drobo-symbolic" : "device-notifier"
    Plasmoid.status: visibleDeviceCount() > 0 ? PlasmaCore.Types.ActiveStatus : PlasmaCore.Types.PassiveStatus
    toolTipMainText: "Disks & Devices"
    toolTipSubText: visibleDeviceCount() + " devices available"

    P5Support.DataSource { id: hotplug; engine: "hotplug"; connectedSources: hotplug.sources }
    P5Support.DataSource { id: solid; engine: "soliddevice"; connectedSources: hotplug.sources }
    P5Support.DataSource {
        id: droboStatus
        engine: "executable"
        connectedSources: ["/usr/local/bin/drobo status"]
        interval: 5000
        onNewData: (sourceName, data) => {
            if (data["exit code"] !== 0) return
            const out = data.stdout || ""
            const cap = out.match(/Capacity:\s+[^/]+\/\s+([0-9.]+\s+TB) usable/)
            const free = out.match(/Free:\s+([0-9.]+\s+TB)/)
            const health = out.match(/Health:\s+([^\n]+)/)
            if (cap) root.droboUsable = cap[1]
            if (free) root.droboFree = free[1]
            if (health) root.droboHealth = health[1].trim()
        }
    }

    function info(udi) { return solid.data[udi] || ({}) }
    function hp(udi) { return hotplug.data[udi] || ({}) }
    function isDrobo(udi) { const s=info(udi); return s["Vendor"] === "Drobo" && s["Product"] === "5D" }
    function includeDevice(udi) {
        const s=info(udi), h=hp(udi)
        if (!h.text) return false
        if (isDrobo(udi)) return true
        const type=s["Type Description"] || ""
        if (type === "Storage Volume") return !!s["Accessible"] || !!s["Hotpluggable"] || !!s["Removable"]
        return !!s["Hotpluggable"] || type === "Camera" || type === "Portable Media Player"
    }
    function category(udi) {
        const t=info(udi)["Type Description"] || ""
        if (t === "Camera") return "Camera"
        if (t === "Portable Media Player") return "Portable Media Player"
        if (t === "Storage Volume") return "Storage Volume"
        return "Other Devices"
    }
    function countCategory(cat) {
        let n=0
        for (let i=0;i<hotplug.sources.length;i++) { const u=hotplug.sources[i]; if (includeDevice(u) && category(u)===cat) n++ }
        return n
    }
    function hasDrobo() {
        for (let i=0;i<hotplug.sources.length;i++) if (isDrobo(hotplug.sources[i])) return true
        return false
    }
    function visibleDeviceCount() {
        let n=0
        for (let i=0;i<hotplug.sources.length;i++) if (includeDevice(hotplug.sources[i])) n++
        return n
    }
    function runSolid(udi, operationName) {
        const service=solid.serviceForSource(udi), op=service.operationDescription(operationName)
        service.startOperationCall(op)
    }
    function runHotplug(udi, predicate) {
        const service=hotplug.serviceForSource(udi), op=service.operationDescription("invokeAction")
        op.predicate=predicate
        service.startOperationCall(op)
    }
    function primaryText(udi) {
        const s=info(udi), actions=hp(udi).actions || []
        if (s["Type Description"] === "Storage Volume") return s["Accessible"] ? "Safely remove" : "Open in File Manager"
        return actions.length ? actions[0].text : "Open"
    }
    function primaryIcon(udi) {
        const s=info(udi), actions=hp(udi).actions || []
        if (s["Type Description"] === "Storage Volume") return s["Accessible"] ? "media-eject" : "system-file-manager"
        return actions.length ? actions[0].icon : "document-open"
    }
    function primaryAction(udi) {
        const s=info(udi), actions=hp(udi).actions || []
        if (s["Type Description"] === "Storage Volume") {
            if (s["Accessible"]) runSolid(udi,"unmount")
            else {
                let predicate=""
                for (let i=0;i<actions.length;i++) if (actions[i].predicate === "openWithFileManager.desktop") predicate=actions[i].predicate
                if (predicate) runHotplug(udi,predicate); else runSolid(udi,"mount")
            }
        } else if (actions.length) runHotplug(udi,actions[0].predicate)
    }

    fullRepresentation: PlasmaExtras.Representation {
        Layout.minimumWidth: Kirigami.Units.gridUnit * 24
        Layout.minimumHeight: Kirigami.Units.gridUnit * 18
        Layout.maximumWidth: Kirigami.Units.gridUnit * 34
        Layout.maximumHeight: Kirigami.Units.gridUnit * 40
        focus: true
        collapseMarginsHint: true

        Flickable {
            id: flick
            anchors.fill: parent
            clip: true
            contentWidth: width
            contentHeight: content.implicitHeight
            boundsBehavior: Flickable.StopAtBounds

            Column {
                id: content
                width: flick.width
                spacing: Kirigami.Units.smallSpacing
                DeviceSection { categoryName: "Camera" }
                DeviceSection { categoryName: "Portable Media Player" }
                DeviceSection { categoryName: "Storage Volume" }
                DeviceSection { categoryName: "Other Devices" }
                PlasmaExtras.PlaceholderMessage {
                    width: parent.width
                    visible: root.visibleDeviceCount() === 0
                    iconName: "drive-removable-media-symbolic"
                    text: "No devices available"
                }
            }
            QQC2.ScrollBar.vertical: QQC2.ScrollBar {}
        }
    }

    component DeviceSection: Column {
        id: section
        required property string categoryName
        width: parent.width
        visible: root.countCategory(categoryName) > 0
        spacing: 0
        PlasmaExtras.ListSectionHeader { width: parent.width; text: section.categoryName }
        Repeater {
            model: hotplug.sources
            delegate: DeviceRow {
                required property string modelData
                udi: modelData
                visible: root.includeDevice(udi) && root.category(udi) === section.categoryName
                width: section.width
            }
        }
    }

    component DeviceRow: Item {
        id: row
        required property string udi
        readonly property var s: root.info(udi)
        readonly property var h: root.hp(udi)
        readonly property bool drobo: root.isDrobo(udi)
        readonly property var actions: h.actions || []
        property bool expanded: false
        implicitHeight: visible ? body.implicitHeight + Kirigami.Units.smallSpacing * 2 : 0

        Rectangle { anchors.fill: parent; color: hover.hovered ? Qt.rgba(1,1,1,0.04) : "transparent"; radius: Kirigami.Units.smallSpacing }
        HoverHandler { id: hover }

        ColumnLayout {
            id: body
            anchors.left: parent.left; anchors.right: parent.right
            anchors.margins: Kirigami.Units.smallSpacing
            spacing: Kirigami.Units.smallSpacing

            RowLayout {
                Layout.fillWidth: true
                spacing: Kirigami.Units.smallSpacing
                Kirigami.Icon {
                    source: row.drobo ? "drobo" : (row.h.icon || row.s["Icon"] || "drive-removable-media")
                    Layout.preferredWidth: Kirigami.Units.iconSizes.medium
                    Layout.preferredHeight: Kirigami.Units.iconSizes.medium
                }
                ColumnLayout {
                    Layout.fillWidth: true; spacing: 0
                    PlasmaComponents3.Label { Layout.fillWidth: true; text: row.drobo ? "Drobo 5D" : (row.h.text || row.s["Description"] || "Device"); elide: Text.ElideRight }
                    PlasmaComponents3.Label {
                        Layout.fillWidth: true
                        opacity: 0.75
                        font.pixelSize: Kirigami.Theme.smallFont.pixelSize
                        text: {
                            if (row.drobo) return root.droboFree + " free of " + root.droboUsable + " usable" + (root.droboHealth ? " · " + root.droboHealth : "")
                            if (row.s["Free Space Text"] && row.s["Size Text"]) return row.s["Free Space Text"] + " free of " + row.s["Size Text"]
                            return ""
                        }
                        elide: Text.ElideRight
                    }
                }
                PlasmaComponents3.ToolButton { icon.name: root.primaryIcon(row.udi); text: root.primaryText(row.udi); onClicked: root.primaryAction(row.udi) }
                PlasmaComponents3.ToolButton { visible: row.actions.length > 0; icon.name: row.expanded ? "arrow-up" : "arrow-down"; onClicked: row.expanded=!row.expanded }
            }
            ColumnLayout {
                Layout.fillWidth: true
                visible: row.expanded
                Repeater {
                    model: row.actions
                    delegate: PlasmaComponents3.ToolButton {
                        required property var modelData
                        Layout.fillWidth: true
                        text: modelData.text
                        icon.name: modelData.icon
                        onClicked: root.runHotplug(row.udi,modelData.predicate)
                    }
                }
            }
        }
    }
}
