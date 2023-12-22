-- protocol definition and dissector for goverlay message headers
--
local goverlay_proto = Proto("goverlay","Goverlay Protocol")

local types = {
    [0] = "Invalid",
    [1] = "Data",
    [2] = "Control",
    [3] = "Test",
    [4] = "Handshake"
}

local subtypes = {
    [0] = "None",
    [1] = "Protobuf",
    [2] = "Request",
    [3] = "Reply",
    [4] = "Keepalive"
}
-- protocol fields
local pf_version = ProtoField.new("Version", "goverlay.version", ftypes.UINT8)
local pf_type = ProtoField.new("Message Type", "goverlay.type", ftypes.UINT8, types)
local pf_subtype = ProtoField.new("Message SubType", "goverlay.subtype", ftypes.UINT8, subtypes)
local pf_reserved = ProtoField.new("Reserved", "goverlay.reserved", ftypes.UINT8, nil, base.HEX)

-- register protofields
goverlay_proto.fields = {pf_version, pf_type, pf_subtype, pf_reserved}

-- field to pull type info for checking payload
local t = Field.new("goverlay.type")

local function getTypeValue(typename)
    for k,v in pairs(types) do
        if v == typename then
            return k
        end
    end
    return nil
end

local function isData()
    local typeinfo = t()
    -- get index of "Data" value in table
    local val = getTypeValue("Data")
    if val == typeinfo() then
        return true
    end
    return false
end
-- dissector function
function goverlay_proto.dissector(buffer,pinfo,tree)
    pinfo.cols.protocol:set("GOVERLAY")
    local pktlen = buffer:reported_length_remaining()

    local root = tree:add(goverlay_proto, buffer:range(0, pktlen))

    root:add(pf_version, buffer:range(0, 1))
    root:add(pf_type, buffer:range(1, 1))
    root:add(pf_subtype, buffer:range(2,1))
    root:add(pf_reserved, buffer:range(3,1))

    local payload_range = buffer:range(4)
    local payload_tree = root:add("Payload:")
    if isData() then
        local ip_dissector = Dissector.get("ip")
        ip_dissector:call(buffer(4):tvb(), pinfo, payload_tree)
        pinfo.cols.protocol:set("GOVERLAY")
    else
        payload_tree:add("Control Message Payload: " .. payload_range)
    end

    -- local subtree = :add(goverlay_proto,buffer(),"Goverlay Protocol Data")
    -- subtree:add(buffer(0,1), "Version: " .. buffer(0,1))
    -- subtree:add(buffer(1,1), "Type: " .. buffer(1,1))
    -- subtree:add(buffer(2,1), "SubType: " .. buffer(2,1))
    -- subtree:add(buffer(3,1), "Reserved: " .. buffer(3,1))
    -- subtree:add(buffer(4), "Payload: " .. buffer(4))
end

-- load the udp.port table
udp_table = DissectorTable.get("udp.port")
-- register port for goverlay protocol
udp_table:add(5555,goverlay_proto)