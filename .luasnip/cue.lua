local ls = require("luasnip")
local s = ls.snippet
local t = ls.text_node
local i = ls.insert_node
local c = ls.choice_node

return {
  -- #Endpoint - HTTP API surface
  s("endpoint", {
    t("endpoint: {"),
    t({ "", "\tverb: " }), c(1, {
      t('"POST"'),
      t('"GET"'),
      t('"PUT"'),
      t('"PATCH"'),
      t('"DELETE"'),
    }),
    t({ "", "\tparams: {" }), i(2), t("}"),
    t({ "", "\tbody: {" }), i(3), t("}"),
    t({ "", '\tpath: "/' }), i(4), t('"'),
    t({ "", "}" }),
  }),

  -- #GetEndpoint - GET-only variant (no body)
  s("getendpoint", {
    t("endpoint: {"),
    t({ "", '\tverb: "GET"' }),
    t({ "", "\tparams: {" }), i(1), t("}"),
    t({ "", '\tpath: "/' }), i(2), t('"'),
    t({ "", "}" }),
  }),

  -- #ChangeSlice - command that emits events
  s("changeslice", {
    i(1, "SliceName"), t(": schema.#ChangeSlice & {"),
    t({ "", '\tname:  "' }), i(2, "SliceName"), t('"'),
    t({ "", "\tactor: _actors." }), i(3, "User"),
    t({ "", "\ttrigger: schema.#EndpointTrigger & {" }),
    t({ "", "\t\tendpoint: {" }),
    t({ "", "\t\t\tverb: " }), c(4, {
      t('"POST"'),
      t('"PUT"'),
      t('"PATCH"'),
      t('"DELETE"'),
    }),
    t({ "", "\t\t\tparams: {" }), i(5), t("}"),
    t({ "", "\t\t\tbody: {" }), i(6), t("}"),
    t({ "", '\t\t\tpath: "/' }), i(7), t('"'),
    t({ "", "\t\t}" }),
    t({ "", "\t}" }),
    t({ "", "\tcommand: {" }),
    t({ "", "\t\tfields: {" }), i(8), t("}"),
    t({ "", "\t\tquery: {" }),
    t({ "", "\t\t\titems: [{" }),
    t({ "", "\t\t\t\ttypes: [" }), i(9), t("]"),
    t({ "", "\t\t\t\ttags: [" }), i(10), t("]"),
    t({ "", "\t\t\t}]" }),
    t({ "", "\t\t}" }),
    t({ "", "\t}" }),
    t({ "", "\temits: [" }), i(11), t("]"),
    t({ "", "\tscenarios: []" }),
    t({ "", "}" }),
  }),

  -- #ViewSlice - query that reads events
  s("viewslice", {
    i(1, "ViewName"), t(": schema.#ViewSlice & {"),
    t({ "", '\tname:  "' }), i(2, "ViewName"), t('"'),
    t({ "", "\tactor: _actors." }), i(3, "User"),
    t({ "", "\tendpoint: schema.#Endpoint & {" }),
    t({ "", '\t\tverb: "GET"' }),
    t({ "", "\t\tparams: {" }), i(4), t("}"),
    t({ "", '\t\tpath: "/' }), i(5), t('"'),
    t({ "", "\t}" }),
    t({ "", "\tquery: {" }),
    t({ "", "\t\titems: [{" }),
    t({ "", "\t\t\ttypes: [" }), i(6), t("]"),
    t({ "", "\t\t\ttags: [" }), i(7), t("]"),
    t({ "", "\t\t}]" }),
    t({ "", "\t}" }),
    t({ "", "\treadModel: schema.#ReadModel & {" }),
    t({ "", '\t\tname: "' }), i(8, "ViewModel"), t('"'),
    t({ "", "\t\tcardinality: " }), c(9, { t('"single"'), t('"table"') }),
    t({ "", "\t\tfields: {" }), i(10), t("}"),
    t({ "", "\t\tmapping: {" }), i(11), t("}"),
    t({ "", "\t}" }),
    t({ "", "\tscenarios: []" }),
    t({ "", "}" }),
  }),

  -- #Event - domain event
  s("event", {
    i(1, "EventName"), t(": {"),
    t({ "", "\tfields: {" }), i(2), t("}"),
    t({ "", "\ttags: [" }), i(3), t("]"),
    t({ "", "}" }),
  }),

  -- #Tag - DCB partitioning tag
  s("tag", {
    i(1, "tag_name"), t(": {param: \""), i(2, "paramName"), t("\"}"),
  }),

  -- #Actor
  s("actor", {
    i(1, "ActorName"), t(": {}"),
  }),

  -- #GWTSuccess - success scenario
  s("gwtsuccess", {
    t("{"),
    t({ "", '\tname: "' }), i(1, "scenario name"), t('"'),
    t({ "", "\tgiven: [" }), i(2), t("]"),
    t({ "", "\twhen: {" }), i(3), t("}"),
    t({ "", "\tthen: {" }),
    t({ "", "\t\tsuccess: true" }),
    t({ "", "\t\tevents: [" }), i(4), t("]"),
    t({ "", "\t}" }),
    t({ "", "}" }),
  }),

  -- #GWTError - error scenario
  s("gwterror", {
    t("{"),
    t({ "", '\tname: "' }), i(1, "error scenario"), t('"'),
    t({ "", "\tgiven: [" }), i(2), t("]"),
    t({ "", "\twhen: {" }), i(3), t("}"),
    t({ "", "\tthen: {" }),
    t({ "", "\t\tsuccess: false" }),
    t({ "", '\t\terror: "' }), i(4), t('"'),
    t({ "", "\t}" }),
    t({ "", "}" }),
  }),

  -- #StoryStep - narrative reference
  s("storystep", {
    t("{"),
    t({ "", '\tkind: "story"' }),
    t({ "", '\tname: "' }), i(1, "step name"), t('"'),
    t({ "", '\tsliceRef: "' }), i(2, "SliceName"), t('"'),
    t({ "", '\tdescription: "' }), i(3), t('"'),
    t({ "", "}" }),
  }),

  -- #QueryItem - DCB query item
  s("queryitem", {
    t("{"),
    t({ "", "\ttypes: [" }), i(1), t("]"),
    t({ "", "\ttags: [{tag: _tags." }), i(2), t(", value: "), i(3), t("}]"),
    t({ "", "}" }),
  }),

  -- #TagRef - tag reference with value (for queries)
  s("tagref", {
    t("{tag: _tags."), i(1), t(", value: "), i(2), t("}"),
  }),

  -- #EventInstance - event with values
  s("eventinstance", {
    t("{event: _events."), i(1), t(", values: {"), i(2), t("}}"),
  }),
}
